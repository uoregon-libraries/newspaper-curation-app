package devtool

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// layout is where a stack keeps things on the host, according to its compose
// files (e.g., the volumes compose.override.yml binds to host directories).
// Nothing here is assumed: if an override binds the workflow tree somewhere
// else, or doesn't bind it at all, the layout says so.
type layout struct {
	// binds maps each volume bound to a host directory (e.g., "news") to that
	// directory
	binds map[string]string
	// logDirs are the directories services write their logs to
	logDirs []string
}

// composeConfig is the part of "compose config" output we care about
type composeConfig struct {
	Services map[string]struct {
		Logging struct {
			Options map[string]string `yaml:"options"`
		} `yaml:"logging"`
	} `yaml:"services"`
	Volumes map[string]*struct {
		DriverOpts map[string]string `yaml:"driver_opts"`
	} `yaml:"volumes"`
}

// parseLayout reads a layout from "compose config" output
func parseLayout(data []byte) (*layout, error) {
	var cfg composeConfig
	var err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("reading compose config: %w", err)
	}

	var l = &layout{binds: make(map[string]string)}
	for name, vol := range cfg.Volumes {
		if vol == nil {
			continue
		}
		var dev = vol.DriverOpts["device"]
		if dev != "" && strings.Contains(vol.DriverOpts["o"], "bind") {
			l.binds[name] = filepath.Clean(dev)
		}
	}

	var seen = make(map[string]bool)
	for _, svc := range cfg.Services {
		var p = svc.Logging.Options["path"]
		if p == "" {
			continue
		}
		var dir = filepath.Dir(p)
		if !seen[dir] {
			seen[dir] = true
			l.logDirs = append(l.logDirs, dir)
		}
	}
	sort.Strings(l.logDirs)
	return l, nil
}

// layout returns the stack's layout, reading it from compose the first time
func (s *Stack) layout() (*layout, error) {
	if s.lay != nil {
		return s.lay, nil
	}

	var out, err = s.composeOutput("config")
	if err != nil {
		return nil, err
	}
	s.lay, err = parseLayout([]byte(out))
	return s.lay, err
}

// hostDirs returns the directories that must exist before the stack can
// start: every bind-mounted volume's directory, the log directories, and
// the directories NCA's settings expect in the workflow tree (if it's bound
// to a host directory)
func (l *layout) hostDirs() []string {
	var dirs []string
	for _, dev := range l.binds {
		dirs = append(dirs, dev)
	}
	if news := l.binds["news"]; news != "" {
		dirs = append(dirs, subdirs(news, workflowSubdirs)...)
	}
	dirs = append(dirs, l.logDirs...)
	sort.Strings(dirs)
	return dirs
}

// savedBinds returns the bind-mounted volumes that hold the stack's state,
// sorted by name: everything but skipVolumes and the issue cache (it's
// rebuildable, just slow), and leaving out any volume whose directory is
// inside another's (e.g., batches, in the workflow tree), since saving the
// outer one saves it, too.
func (l *layout) savedBinds() []string {
	var names []string
	for name, dev := range l.binds {
		if name == "issue-cache" || isSkipped(name) || l.nested(dev) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// nested reports whether dir is inside another bind's directory
func (l *layout) nested(dir string) bool {
	for _, other := range l.binds {
		if other != dir && strings.HasPrefix(dir, other+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
