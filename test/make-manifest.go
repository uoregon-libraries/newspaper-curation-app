package main

import (
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil/manifest"
	"github.com/uoregon-libraries/gopkg/logger"
)

var l = logger.New(logger.Debug, false)

func main() {
	if len(os.Args) < 2 {
		l.Fatalf("You must specify a path to make a manifest")
	}
	var pth = os.Args[1]
	makeManifest(pth)
}

func makeManifest(pth string) {
	var m = manifest.New(pth)
	var err = m.Build()
	if err != nil {
		l.Fatalf("Unable to build manifest for %q: %s", pth, err)
	}
	err = m.Write()
	if err != nil {
		l.Fatalf("Unable to write manifest for %q: %s", pth, err)
	}
}
