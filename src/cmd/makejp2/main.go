// This script runs the jp2 transformer against a single PDF or TIFF

package main

import (
	"config"
	"derivatives/jp2"

	"fmt"
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

func usageFail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "usage: makejp2 <source> <jp2 output file> <config file>")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 4 {
		usageFail("Too few arguments")
	}
	if len(os.Args) > 4 {
		usageFail("Too many arguments")
	}

	var sourceFile = os.Args[1]
	var jp2File = os.Args[2]
	var config, err = config.Parse(os.Args[3])
	if err != nil {
		usageFail("Unable to read config file %q: %s", os.Args[3], err)
	}

	if !fileutil.Exists(sourceFile) {
		usageFail("%q does not exist", sourceFile)
	}

	var t = jp2.New(sourceFile, "ONE-OFF", jp2File, config.Quality, config.DPI)
	t.GhostScript = config.GhostScript
	t.OPJCompress = config.OPJCompress
	t.OPJDecompress = config.OPJDecompress
	err = t.Transform()
	if err != nil {
		fmt.Printf("Unable to generate a JP2: %s\n", err)
	}
}
