// Package devtool drives NCA's compose stacks for development and testing:
// the dev stack, configured by .env and compose.override.yml, and the test
// stack, a separate copy configured entirely by the repository so tests don't
// depend on anybody's dev setup.
package devtool

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Stack is one of NCA's compose stacks, and how to run compose on it
type Stack struct {
	// Name is "dev" or "test", for messages
	Name string
	// Root is the NCA checkout
	Root string

	project     string
	composeArgs []string
	env         []string

	// newsDir is the host directory bound to the "news" volume, NCA's workflow
	// tree. hostDirs must exist before the stack can start.
	newsDir  string
	hostDirs []string

	savepointDir string

	// publicHost is the host in the stack's URLs, for messages
	publicHost string

	// cacheDir is the host directory bound to the "issue-cache" volume. If
	// it's empty when the stack starts, it's seeded from seedCacheFrom.
	cacheDir      string
	seedCacheFrom string

	// Out and Err are where compose and everything it runs write
	Out io.Writer
	Err io.Writer
}

// workflowSubdirs are the directories NCA's settings expect in the news
// volume
var workflowSubdirs = []string{
	"backup/originals", "batch-archive", "errors", "outgoing", "page-review",
	"production-batches", "scans", "sftp", "workflow",
}

// NewDevStack returns the dev stack: compose.yml and compose.override.yml,
// configured by .env, with dev volumes in DEV-VOLS
func NewDevStack() (*Stack, error) {
	var root, err = findRoot()
	if err != nil {
		return nil, err
	}

	var envPath = filepath.Join(root, ".env")
	var vars []envVar
	_, vars, err = readEnvFile(envPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("no .env file: copy .env.example to .env (see the Developer's Guide)")
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", envPath, err)
	}

	// Like compose, the environment wins over .env
	var dotenv = resolveEnv(vars, root)
	var get = func(k string) string {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
		return dotenv[k]
	}

	var s = &Stack{
		Name:         "dev",
		Root:         root,
		project:      get("COMPOSE_PROJECT_NAME"),
		newsDir:      filepath.Join(root, "DEV-VOLS", "news"),
		savepointDir: filepath.Join(root, "DEV-VOLS", "savepoints"),
		cacheDir:     filepath.Join(root, "DEV-VOLS", "issue-cache"),
		env:          setEnv(os.Environ(), "PWD", root),
		Out:          os.Stdout,
		Err:          os.Stderr,
	}
	if s.project == "" {
		s.project = filepath.Base(root)
	}
	s.hostDirs = append(subdirs(s.newsDir, workflowSubdirs), s.cacheDir)
	var logDir = get("CONTAINER_LOG_DIR")
	if logDir != "" {
		s.hostDirs = append(s.hostDirs, logDir)
	}

	s.composeArgs = append([]string{"-p", s.project}, profileArgs(get("COMPOSE_PROFILES"))...)
	return s, nil
}

// The test stack's compose project, profiles, and files
const (
	testProject  = "nca-test"
	testProfiles = "local-dev,sftpgo"
	testOverride = "test/stack/compose.yml"
	testEnvFile  = "test/var/stack.env"
)

// testEnv is the test stack's configuration: .env.example with these values
// (and PUBLIC_HOST; see testHost) replaced. Settings the test recipes depend
// on are pinned here, even when they match .env.example, so changing dev's
// defaults can't change test results. Ports differ from dev's so both stacks
// can run at once.
var testEnv = []envVar{
	{"COMPOSE_PROJECT_NAME", testProject},
	{"COMPOSE_PROFILES", testProfiles},
	{"CONTAINER_LOG_DIR", "${PWD}/test/var/logs"},
	{"PUBLIC_SCHEME", "http"},
	{"PUBLIC_PORT", "4333"},
	{"PUBLIC_PORT_SUFFIX", ":${PUBLIC_PORT}"},
	{"NCA_URL_PATH", "/nca"},
	{"ONI_PROD_PORT", "4080"},
	{"ONI_STAGING_PORT", "4082"},
	{"SFTPGO_WEB_PORT", "4081"},
	{"SFTPGO_SFTP_PORT", "4022"},
	{"NCA_DEBUG", "true"},
	{"PDF_BATCH_MARC_ORG_CODE", "oru"},
	{"MIN_BATCH_SIZE", "4"},
	{"MAX_BATCH_SIZE", "4000"},
	{"DURATION_ISSUE_CONSIDERED_DANGEROUS", "48h"},
	{"DURATION_ISSUE_CONSIDERED_NEW", "336h"},
}

// testHost returns the host the test stack's URLs use: PUBLIC_HOST from
// .env if it's set there, else localhost. It's the one dev setting the test
// stack uses, because it describes where your browser is (e.g., when NCA runs
// on a VM), not how the stack behaves.
func testHost(root string) string {
	var _, vars, err = readEnvFile(filepath.Join(root, ".env"))
	if err != nil {
		return "localhost"
	}
	var host = resolveEnv(vars, root)["PUBLIC_HOST"]
	if host == "" {
		return "localhost"
	}
	return host
}

// testURL returns the test stack's URL for the given port setting, e.g.,
// testURL("ONI_PROD_PORT")
func (s *Stack) testURL(portKey string) string {
	var port string
	for _, v := range testEnv {
		if v.key == portKey {
			port = v.value
		}
	}
	return "http://" + s.publicHost + ":" + port
}

// NewTestStack returns the test stack: compose.yml and test/stack/compose.yml,
// configured by .env.example plus testEnv, with its workflow tree in
// test/fakemount and everything else in test/var. Nothing in it comes from
// compose.override.yml or the environment, and only PUBLIC_HOST comes from
// .env.
func NewTestStack() (*Stack, error) {
	var root, err = findRoot()
	if err != nil {
		return nil, err
	}

	var host = testHost(root)
	var envPath = filepath.Join(root, testEnvFile)
	err = writeTestEnv(root, envPath, append([]envVar{{"PUBLIC_HOST", host}}, testEnv...))
	if err != nil {
		return nil, err
	}

	var env []string
	env, err = testStackEnviron(root, envPath)
	if err != nil {
		return nil, err
	}

	var s = &Stack{
		Name:          "test",
		Root:          root,
		project:       testProject,
		newsDir:       filepath.Join(root, "test", "fakemount"),
		savepointDir:  filepath.Join(root, "test", "var", "savepoints"),
		cacheDir:      filepath.Join(root, "test", "var", "issue-cache"),
		seedCacheFrom: filepath.Join(root, "DEV-VOLS", "issue-cache"),
		publicHost:    host,
		env:           env,
		Out:           os.Stdout,
		Err:           os.Stderr,
	}
	s.hostDirs = append(subdirs(s.newsDir, workflowSubdirs), s.cacheDir, filepath.Join(root, "test", "var", "logs"))
	s.composeArgs = append([]string{
		"--env-file", testEnvFile,
		"-f", "compose.yml",
		"-f", testOverride,
		"-p", testProject,
	}, profileArgs(testProfiles)...)
	return s, nil
}

// writeTestEnv writes the test stack's env file: .env.example with
// overrides replacing its values
func writeTestEnv(root, path string, overrides []envVar) error {
	var lines, _, err = readEnvFile(filepath.Join(root, ".env.example"))
	if err != nil {
		return fmt.Errorf("reading .env.example: %w", err)
	}

	var out = []string{
		"# Generated by ncatest from .env.example, with the test stack's own",
		"# settings (see tools/ncadev). Don't edit: it's rewritten every run.",
		"",
	}
	out = append(out, mergeEnv(lines, overrides)...)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err == nil {
		err = os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0644)
	}
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

var composeVarRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)`)

// testStackEnviron returns our environment minus anything that could change
// the test stack's configuration: compose's own variables, everything the env
// file sets, and everything the compose files refer to. Variables in the
// environment win over the env file, so leaving any of them in would let a
// developer's shell change the test stack.
func testStackEnviron(root, envPath string) ([]string, error) {
	var drop = make(map[string]bool)
	var _, vars, err = readEnvFile(envPath)
	if err != nil {
		return nil, err
	}
	for _, v := range vars {
		drop[v.key] = true
	}

	for _, f := range []string{"compose.yml", testOverride} {
		var data []byte
		data, err = os.ReadFile(filepath.Join(root, f))
		if err != nil {
			return nil, err
		}
		for _, m := range composeVarRef.FindAllStringSubmatch(string(data), -1) {
			drop[m[1]] = true
		}
	}

	var env []string
	for _, kv := range os.Environ() {
		var k, _, _ = strings.Cut(kv, "=")
		if drop[k] || strings.HasPrefix(k, "COMPOSE_") {
			continue
		}
		env = append(env, kv)
	}
	return setEnv(env, "PWD", root), nil
}

// findRoot returns the NCA checkout containing the current directory
func findRoot() (string, error) {
	var dir, err = os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if isNCARoot(dir) {
			return dir, nil
		}
		var parent = filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("not in an NCA checkout: run this from the NCA directory (or below it)")
		}
		dir = parent
	}
}

func isNCARoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "compose.yml")); err != nil {
		return false
	}
	var f, err = os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	defer f.Close()

	var scanner = bufio.NewScanner(f)
	return scanner.Scan() && strings.TrimSpace(scanner.Text()) == "module github.com/uoregon-libraries/newspaper-curation-app"
}

func subdirs(base string, names []string) []string {
	var dirs = make([]string, len(names))
	for i, name := range names {
		dirs[i] = filepath.Join(base, name)
	}
	return dirs
}

// profileArgs turns a comma-separated profile list into --profile flags.
// podman-compose before 1.6 ignores COMPOSE_PROFILES (see scripts/compose).
func profileArgs(profiles string) []string {
	var args []string
	for _, p := range strings.Split(profiles, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			args = append(args, "--profile", p)
		}
	}
	return args
}

// setEnv returns env with key set to val, replacing any existing value
func setEnv(env []string, key, val string) []string {
	var out = make([]string, 0, len(env)+1)
	for _, kv := range env {
		if !strings.HasPrefix(kv, key+"=") {
			out = append(out, kv)
		}
	}
	return append(out, key+"="+val)
}

// quietPodman turns off the banner "podman compose" prints every time it runs
// podman-compose, unless it's been set already
func quietPodman(env []string) []string {
	for _, kv := range env {
		if strings.HasPrefix(kv, "PODMAN_COMPOSE_WARNING_LOGS=") {
			return env
		}
	}
	return append(env, "PODMAN_COMPOSE_WARNING_LOGS=false")
}

// command returns a command that runs in the checkout, with the stack's
// environment and output
func (s *Stack) command(name string, args ...string) *exec.Cmd {
	var cmd = exec.Command(name, args...)
	cmd.Dir = s.Root
	cmd.Env = quietPodman(s.env)
	cmd.Stdin = os.Stdin
	cmd.Stdout = s.Out
	cmd.Stderr = s.Err
	return cmd
}

// composeCmd returns a "podman compose" command for this stack
func (s *Stack) composeCmd(args ...string) *exec.Cmd {
	var full = append([]string{"compose"}, s.composeArgs...)
	return s.command("podman", append(full, args...)...)
}

// Compose runs compose on this stack, e.g., Compose("ps")
func (s *Stack) Compose(args ...string) error {
	return run(s.composeCmd(args...))
}

// composeOutput runs compose and returns what it wrote to stdout
func (s *Stack) composeOutput(args ...string) (string, error) {
	var cmd = s.composeCmd(args...)
	var buf bytes.Buffer
	cmd.Stdin = nil
	cmd.Stdout = &buf
	var err = run(cmd)
	return buf.String(), err
}

// podman runs podman directly, e.g., for volume commands
func (s *Stack) podman(args ...string) error {
	return run(s.command("podman", args...))
}

// podmanOutput runs podman and returns what it wrote to stdout, trimmed
func (s *Stack) podmanOutput(args ...string) (string, error) {
	var cmd = s.command("podman", args...)
	var buf bytes.Buffer
	cmd.Stdin = nil
	cmd.Stdout = &buf
	var err = run(cmd)
	return strings.TrimSpace(buf.String()), err
}

// commandError is a command that failed. Its message names the command, but
// the error it wraps is what exec returned, so callers can still get the
// exit code.
type commandError struct {
	cmd string
	err error
}

func (e *commandError) Error() string {
	return e.cmd + ": " + e.err.Error()
}

func (e *commandError) Unwrap() error {
	return e.err
}

func run(cmd *exec.Cmd) error {
	var err = cmd.Run()
	if err == nil {
		return nil
	}

	// Name the command without its global compose flags, which are long and
	// the same for every compose command
	var args = cmd.Args
	if len(args) > 2 && args[1] == "compose" {
		args = append([]string{"podman", "compose"}, composeSubcommand(args[2:])...)
	}
	return &commandError{cmd: strings.Join(args, " "), err: err}
}

// composeSubcommand strips global flags (and their values) from compose args
func composeSubcommand(args []string) []string {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "-p", "--env-file", "--profile":
			i++
		default:
			return args[i:]
		}
	}
	return nil
}

// isTerminal reports whether f is a terminal (well, a character device,
// which is close enough here)
func isTerminal(f *os.File) bool {
	var info, err = f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
