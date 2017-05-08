package legacyfinder

import (
	"config"
	"issuefinder"
)

type Scanner struct {
	*Finder
}

// NewScanner creates a new single-use scanner
func NewScanner(conf *config.Config, webroot, tempdir string) *Scanner {
	return &Scanner{Finder: &Finder{finder: issuefinder.New(), config: conf, webroot: webroot, tempdir: tempdir}}
}
