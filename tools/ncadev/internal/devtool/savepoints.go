package devtool

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// skipVolumes aren't the stack's state: Go's caches and the dev image's
// compiled binaries
var skipVolumes = []string{"go-cache", "dev-bin"}

// dataVolumes returns the compose names (e.g., "db") of the stack's volumes
// that hold its state: everything compose created except skipVolumes and
// volumes bound to host directories. The workflow tree is saved separately,
// and the issue cache isn't saved at all.
func (s *Stack) dataVolumes() ([]string, error) {
	var out, err = s.podmanOutput("volume", "ls", "-q", "--filter", "label=io.podman.compose.project="+s.project)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, vol := range strings.Fields(out) {
		var name = strings.TrimPrefix(vol, s.project+"_")
		if slices.Contains(skipVolumes, name) {
			continue
		}
		var opt string
		opt, err = s.podmanOutput("volume", "inspect", "--format", `{{index .Options "o"}}`, vol)
		if err != nil {
			return nil, err
		}
		if opt == "bind" {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// savepointPath validates a save point's name and returns its directory
func (s *Stack) savepointPath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("invalid save point name %q", name)
	}
	return filepath.Join(s.savepointDir, name), nil
}

// HasSavepoint reports whether the named save point exists
func (s *Stack) HasSavepoint(name string) bool {
	var dir, err = s.savepointPath(name)
	if err != nil {
		return false
	}
	var info os.FileInfo
	info, err = os.Stat(dir)
	return err == nil && info.IsDir()
}

// Save saves the stack's state (its workflow tree and data volumes) as the
// named save point, replacing any save point with that name. The stack has
// to stop while it's saved, and is left stopped.
func (s *Stack) Save(name string) error {
	var dir, err = s.savepointPath(name)
	if err != nil {
		return err
	}

	// Create every volume, so the save point is complete, then stop
	// everything so nothing changes mid-save
	err = s.prepareHostDirs()
	if err == nil {
		err = s.Compose("up", "--no-start")
	}
	if err == nil {
		err = s.Down()
	}
	if err != nil {
		return err
	}

	err = os.RemoveAll(dir)
	if err == nil {
		err = os.MkdirAll(dir, 0755)
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(s.Out, "Saving the workflow tree (%s)\n", s.newsDir)
	err = s.podman("unshare", "tar", "-cpf", filepath.Join(dir, "news.tar"), "-C", s.newsDir, ".")
	if err != nil {
		return err
	}

	var vols []string
	vols, err = s.dataVolumes()
	if err != nil {
		return err
	}
	for _, vol := range vols {
		fmt.Fprintf(s.Out, "Saving volume %s\n", vol)
		err = s.podman("volume", "export", s.project+"_"+vol, "--output", filepath.Join(dir, "volume-"+vol+".tar"))
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(s.Out, "Saved %s stack to %s\n", s.Name, dir)
	return nil
}

// findSavepoint returns the directory for name: the save point with that
// exact name, or failing that, the only one whose name starts with it
func (s *Stack) findSavepoint(name string) (dir string, exact bool, err error) {
	dir, err = s.savepointPath(name)
	if err != nil {
		return "", false, err
	}
	if s.HasSavepoint(name) {
		return dir, true, nil
	}

	var matches []string
	matches, err = filepath.Glob(dir + "*")
	if err != nil {
		return "", false, err
	}
	switch len(matches) {
	case 0:
		return "", false, fmt.Errorf("no save point named %q in %s", name, s.savepointDir)
	case 1:
		return matches[0], false, nil
	}
	for i := range matches {
		matches[i] = filepath.Base(matches[i])
	}
	return "", false, fmt.Errorf("%q matches more than one save point: %s", name, strings.Join(matches, ", "))
}

// Restore replaces the stack's state with the named save point (or the only
// one whose name starts with name). The stack is left stopped.
func (s *Stack) Restore(name string) error {
	var dir, exact, err = s.findSavepoint(name)
	if err != nil {
		return err
	}
	if !exact {
		fmt.Fprintf(s.Out, "Restoring %s in five seconds (Ctrl-C to cancel)\n", filepath.Base(dir))
		time.Sleep(5 * time.Second)
	}

	err = s.Nuke()
	if err == nil {
		err = os.MkdirAll(s.newsDir, 0755)
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(s.Out, "Restoring the workflow tree (%s)\n", s.newsDir)
	err = s.podman("unshare", "tar", "-xpf", filepath.Join(dir, "news.tar"), "-C", s.newsDir)
	if err != nil {
		return err
	}

	var tarfiles []string
	tarfiles, err = filepath.Glob(filepath.Join(dir, "volume-*.tar"))
	if err != nil {
		return err
	}
	for _, tarfile := range tarfiles {
		var vol = strings.TrimSuffix(strings.TrimPrefix(filepath.Base(tarfile), "volume-"), ".tar")
		fmt.Fprintf(s.Out, "Restoring volume %s\n", vol)
		var full = s.project + "_" + vol
		err = s.podman("volume", "create", "--label", "io.podman.compose.project="+s.project, full)
		if err == nil {
			err = s.podman("volume", "import", full, tarfile)
		}
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(s.Out, "Restored %s stack from %s\n", s.Name, dir)
	return nil
}

// Nuke stops the stack and deletes its state: its data volumes and its
// workflow tree. Go's caches, NCA's issue cache, and save points are kept.
func (s *Stack) Nuke() error {
	var err = s.Down()
	if err != nil {
		return err
	}

	var vols []string
	vols, err = s.dataVolumes()
	if err == nil {
		err = s.removeVolumes(vols...)
	}
	if err != nil {
		return err
	}

	// Containers may have written files we can only delete as root in
	// podman's user namespace
	return s.podman("unshare", "rm", "-rf", s.newsDir)
}
