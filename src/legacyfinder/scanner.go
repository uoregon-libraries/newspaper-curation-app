package legacyfinder

import (
	"config"
	"issuefinder"
)

// A Scanner is just a finder.  I don't recall why this was necessary.
type Scanner struct {
	*Finder
}

// NewScanner creates a new single-use scanner
func NewScanner(conf *config.Config, webroot, tempdir string) *Scanner {
	return &Scanner{Finder: &Finder{finder: issuefinder.New(), config: conf, webroot: webroot, tempdir: tempdir}}
}
