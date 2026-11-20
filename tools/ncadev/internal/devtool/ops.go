package devtool

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// prepareHostDirs creates the host directories the stack's volumes and logs
// need, and seeds the issue cache if it's empty
func (s *Stack) prepareHostDirs() error {
	var lay, err = s.layout()
	if err != nil {
		return err
	}
	for _, dir := range lay.hostDirs() {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	if s.seedCache {
		return s.seedIssueCache(lay.binds["issue-cache"])
	}
	return nil
}

// devCacheDir returns the dev stack's issue cache directory, or "" if there
// isn't one: the directory its issue-cache volume is bound to, or failing
// that, where podman keeps the volume
func (s *Stack) devCacheDir() string {
	var get, err = devEnv(s.Root)
	if err != nil {
		return ""
	}
	var vol = devProject(s.Root, get) + "_issue-cache"
	var cmd = s.command("podman", "volume", "inspect", "--format",
		`{{index .Options "o"}}|{{index .Options "device"}}|{{.Mountpoint}}`, vol)
	cmd.Stdin, cmd.Stderr = nil, nil
	var out []byte
	out, err = cmd.Output()
	if err != nil {
		return ""
	}

	var parts = strings.SplitN(strings.TrimSpace(string(out)), "|", 3)
	if len(parts) != 3 {
		return ""
	}
	if strings.Contains(parts[0], "bind") {
		return parts[1]
	}
	return parts[2]
}

// seedIssueCache copies the dev stack's issue cache into cacheDir if it's
// empty, to skip the long wait (ten minutes or more) while NCA builds a cache
// from the live site. The finder cache is left out: it describes the dev
// stack's workflow tree, not this one's.
func (s *Stack) seedIssueCache(cacheDir string) error {
	if cacheDir == "" {
		return nil
	}
	var entries, err = os.ReadDir(cacheDir)
	if err != nil || len(entries) > 0 {
		return err
	}
	var from = s.devCacheDir()
	if from == "" {
		return nil
	}
	entries, err = os.ReadDir(from)
	if err != nil || len(entries) == 0 {
		return nil
	}

	fmt.Fprintf(s.Out, "Seeding the issue cache from the dev stack's (%s)\n", from)
	return filepath.WalkDir(from, func(pth string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		var rel, _ = filepath.Rel(from, pth)
		var dst = filepath.Join(cacheDir, rel)
		switch {
		case e.Name() == "finder.cache":
			return nil
		case e.IsDir():
			return os.MkdirAll(dst, 0755)
		case e.Type().IsRegular():
			return copyFile(pth, dst)
		}
		return nil
	})
}

func copyFile(src, dst string) error {
	var in, err = os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	var out *os.File
	out, err = os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	var closeErr = out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// Up starts the stack, or the given services, or applies changes to them
func (s *Stack) Up(services ...string) error {
	var err = s.prepareHostDirs()
	if err != nil {
		return err
	}
	return s.Compose(append([]string{"up", "-d"}, services...)...)
}

// Down stops the stack and removes its containers
func (s *Stack) Down() error {
	return s.Compose("down")
}

// Build builds all the stack's images, including the "cli" service's
func (s *Stack) Build() error {
	var err = s.Compose("build")
	if err == nil {
		err = s.Compose("--profile", "tools", "build", "cli")
	}
	return err
}

// ttyFlag returns -T (no TTY) unless we're attached to a terminal, so piping
// and redirection work
func (s *Stack) ttyFlag() []string {
	if s.Out == os.Stdout && isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		return nil
	}
	return []string{"-T"}
}

// ensureCLIImage builds the cli service's image if it doesn't exist. Unlike
// "up", "run" doesn't build a missing image: it tries to pull it instead.
func (s *Stack) ensureCLIImage() error {
	if s.podmanQuiet("image", "exists", "localhost/"+s.project+"_cli:latest") == nil {
		return nil
	}
	return s.Compose("--profile", "tools", "build", "cli")
}

// podmanQuiet runs podman with no output, for checks
func (s *Stack) podmanQuiet(args ...string) error {
	var cmd = s.command("podman", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	return cmd.Run()
}

// cliArgs returns the compose args that run args in a throwaway cli container.
// --no-deps guarantees a one-off command never starts services that were
// stopped on purpose.
func (s *Stack) cliArgs(tty []string, args []string) []string {
	var full = append([]string{"--profile", "tools", "run", "--rm", "--no-deps"}, tty...)
	full = append(full, "cli")
	return append(full, args...)
}

// CLI runs a command in a throwaway NCA container, e.g., CLI("queue-batches").
// In dev images, NCA binaries are rebuilt from the checkout first.
func (s *Stack) CLI(args ...string) error {
	var err = s.ensureCLIImage()
	if err == nil {
		err = s.Compose(s.cliArgs(s.ttyFlag(), args)...)
	}
	return err
}

// cliOutput is CLI, but returns what the command wrote to stdout
func (s *Stack) cliOutput(args ...string) (string, error) {
	var err = s.ensureCLIImage()
	if err != nil {
		return "", err
	}
	return s.composeOutput(s.cliArgs([]string{"-T"}, args)...)
}

// Harness runs a test harness command (see test/harness) in a cli container
func (s *Stack) Harness(args ...string) error {
	return s.CLI(append([]string{"go", "run", "./test/harness"}, args...)...)
}

// SQL runs a MariaDB client on NCA's database, e.g., SQL("-e", "SELECT 1")
func (s *Stack) SQL(args ...string) error {
	var script = `MYSQL_PWD=$NCA_DB_PASSWORD exec mysql -h"$NCA_DB_HOST" -u"$NCA_DB_USER" "$NCA_DB_DATABASE" "$@"`
	return s.CLI(append([]string{"bash", "-c", script, "mysql"}, args...)...)
}

// upExcept starts every service except those listed (and cli, which only
// runs one-off commands)
func (s *Stack) upExcept(exclude ...string) error {
	var out, err = s.composeOutput("config", "--services")
	if err != nil {
		return err
	}

	var services []string
	for _, svc := range strings.Fields(out) {
		if svc != "cli" && !slices.Contains(exclude, svc) {
			services = append(services, svc)
		}
	}
	return s.Up(services...)
}

// upDeps starts everything except NCA itself (and the proxy in front of it),
// for setup that mustn't race the job runner
func (s *Stack) upDeps() error {
	return s.upExcept("web", "workers", "proxy")
}

// resetVolumes hold the databases (NCA's, both ONIs', and SFTPGo's) and
// related state that a reset wipes
var resetVolumes = []string{
	"db", "locks", "sftpgo-home", "sftpgo-backups",
	"oni-staging-data-mariadb", "oni-staging-data-solr",
	"oni-prod-data-mariadb", "oni-prod-data-solr",
}

// Reset wipes NCA's database, both ONIs' databases and Solr indexes, and
// SFTPGo's data, then reseeds them (see the harness's "seed" command) and
// starts the stack. Go's caches and NCA's issue cache are kept.
func (s *Stack) Reset() error {
	var err = s.resetData()
	if err == nil {
		err = s.Up()
	}
	return err
}

// Prep is Reset, plus a fresh copy of the test issues in the workflow tree.
// NCA stays stopped until the end, so the job runner can't start on issues
// while they're being replaced.
func (s *Stack) Prep() error {
	var err = s.prepData()
	if err == nil {
		err = s.Up()
	}
	return err
}

// prepData is Prep, but leaves the stack stopped. Every service using the
// workflow tree needs a consistent view of it, and stale views are painful
// enough that we stop everything once the issues are in place.
func (s *Stack) prepData() error {
	var err = s.resetData()
	if err == nil {
		err = s.loadIssues()
	}
	if err == nil {
		err = s.Down()
	}
	return err
}

// resetData is Reset, but leaves NCA stopped
func (s *Stack) resetData() error {
	var err = s.Down()
	if err == nil {
		err = s.removeVolumes(resetVolumes...)
	}
	if err == nil {
		err = s.upDeps()
	}
	if err == nil {
		err = s.CLI("migrate")
	}
	if err == nil {
		err = s.waitForONIs()
	}
	if err == nil {
		err = s.Harness("seed")
	}
	return err
}

// waitForONIs waits until both ONIs answer HTTP requests. Loading titles
// interacts with Solr in some disastrous way if ONI hasn't finished setting
// it up, and ONI has no simple "are you initialized?" query.
func (s *Stack) waitForONIs() error {
	for _, env := range []string{"staging", "prod"} {
		fmt.Fprintf(s.Out, "Waiting for ONI %s", env)
		var deadline = time.Now().Add(10 * time.Minute)
		for {
			var cmd = s.composeCmd("exec", "-T", "oni-"+env, "curl", "-s", "--head", "localhost")
			cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
			if cmd.Run() == nil {
				break
			}
			if time.Now().After(deadline) {
				fmt.Fprintln(s.Out)
				return fmt.Errorf("ONI %s still isn't answering after ten minutes: check its logs", env)
			}
			fmt.Fprint(s.Out, ".")
			time.Sleep(time.Second)
		}
		fmt.Fprintln(s.Out)
	}
	return nil
}

// loadIssues replaces the workflow tree with the test issues in
// test/sources, then queues the born-digital ones for page review, the way
// a curator would. Scanned issues don't need queueing: NCA picks them up on
// its own.
func (s *Stack) loadIssues() error {
	// Issues are backdated a month: bulk-issue-queue won't touch issues it
	// considers new
	var err = s.Harness("copy-sources")
	if err == nil {
		err = s.Harness("make-older", "--days", "30")
	}
	if err == nil {
		err = s.queueBornDigital()
	}
	return err
}

// lccnPattern matches what bulk-issue-queue lists first on each line for an
// LCCN with born-digital issues. Anything else in its output (e.g., build
// output from the cli container) is ignored.
var lccnPattern = regexp.MustCompile(`^[a-z]{0,3}[0-9]{8,10}$`)

// queueBornDigital queues every born-digital issue, one LCCN at a time,
// since that's how bulk-issue-queue works
func (s *Stack) queueBornDigital() error {
	var out, err = s.cliOutput("bulk-issue-queue", "--type=borndigital")
	if err != nil {
		return err
	}

	for _, line := range strings.Split(out, "\n") {
		var fields = strings.Fields(line)
		if len(fields) == 0 || !lccnPattern.MatchString(fields[0]) {
			continue
		}
		err = s.CLI("bulk-issue-queue", "--type=borndigital", "--key="+fields[0])
		if err != nil {
			return err
		}
	}
	return nil
}

// WorkOnce stops the workers service and runs jobs in the foreground,
// verbosely, until there are none left. The workers service stays stopped.
func (s *Stack) WorkOnce() error {
	var err = s.Compose("stop", "workers")
	if err == nil {
		err = s.CLI("run-jobs", "-v", "--exit-when-done", "watchall")
	}
	return err
}

// InitSQL regenerates docker/init.sql, the migrated schema new databases
// start from, then rebuilds the db image with it. This wipes NCA's database.
func (s *Stack) InitSQL() error {
	var err = s.Down()
	if err == nil {
		err = s.removeVolumes("db", "locks")
	}
	if err == nil {
		err = s.Up("db")
	}
	if err == nil {
		err = s.CLI("migrate")
	}
	if err != nil {
		return err
	}

	var path = filepath.Join(s.Root, "docker", "init.sql")
	var f *os.File
	f, err = os.Create(path)
	if err != nil {
		return err
	}
	var dump = `MYSQL_PWD=$MARIADB_PASSWORD exec mariadb-dump -u"$MARIADB_USER" --databases "$MARIADB_DATABASE" --routines --triggers`
	var cmd = s.composeCmd("exec", "-T", "db", "sh", "-c", dump)
	cmd.Stdin = nil
	cmd.Stdout = f
	err = run(cmd)
	var closeErr = f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = s.Compose("build", "db")
	}
	return err
}

// removeVolumes removes the given volumes (by their compose names, e.g.,
// "db") if they exist
func (s *Stack) removeVolumes(names ...string) error {
	for _, name := range names {
		var vol = s.project + "_" + name
		if s.podmanQuiet("volume", "exists", vol) != nil {
			continue
		}
		var err = s.podman("volume", "rm", vol)
		if err != nil {
			return err
		}
	}
	return nil
}
