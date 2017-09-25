// This script runs the alto transformer against a single PDF

package main

import (
	"derivatives/alto"
	"fileutil"
	"fmt"
	"os"
	"strconv"
)

func usageFail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "usage: pdf-to-alto-xml <pdf file> <output xml file> <pdf dpi> <page number>")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 5 {
		usageFail("Too few arguments")
	}
	if len(os.Args) > 5 {
		usageFail("Too many arguments")
	}

	var pdfFile = os.Args[1]
	var xmlFile = os.Args[2]
	var dpi, _ = strconv.ParseFloat(os.Args[3], 64)
	var pageNumber, _ = strconv.Atoi(os.Args[4])

	if !fileutil.Exists(pdfFile) {
		usageFail("%q does not exist", pdfFile)
	}

	if dpi == 0 {
		usageFail("%q is not a valid DPI value (not a number)", os.Args[3])
	}
	if dpi < 72 {
		usageFail("%d is not a valid DPI value (must be at least 72)", dpi)
	}
	if pageNumber == 0 {
		usageFail("%q is not a valid page number (not a number)", os.Args[4])
	}

	var t = alto.New(pdfFile, xmlFile, dpi, pageNumber)
	var err = t.Transform()
	if err != nil {
		fmt.Printf("Unable to run ALTO transform: %s\n", err)
	}
}
