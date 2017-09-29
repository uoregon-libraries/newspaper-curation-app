package main

import (
	"bytes"
	"logger"
	"os/exec"
	"strconv"
	"strings"
)

// pdfImageDPI holds x and y dpis gathered from PDF images
type pdfImageDPI struct {
	xDPI float64
	yDPI float64
}

// getPDFDPIs returns an array of pdfImageDPIs
func getPDFDPIs(path string) []pdfImageDPI {
	var cmdParts = []string{"pdfimages", "-list", path}
	var cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
	var output, err = cmd.CombinedOutput()
	if err != nil {
		logger.Error(`Failed to run "%s": %s`, strings.Join(cmdParts, " "), err)
		for _, line := range bytes.Split(output, []byte("\n")) {
			logger.Debug("--> %s", line)
		}

		return nil
	}

	var dpis []pdfImageDPI
	for i, line := range bytes.Split(output, []byte("\n")) {
		// The first two lines don't give us any information
		if bytes.HasPrefix(line, []byte("page")) || bytes.HasPrefix(line, []byte("--------")) {
			continue
		}

		// The last line appears to always be blank
		if len(line) == 0 {
			continue
		}

		var parts = bytes.Fields(line)
		if len(parts) < 14 {
			logger.Error("Too few fields in line %d of %q output: %q", i+1, strings.Join(cmdParts, " "), line)
			return nil
		}

		var xdpiString, ydpiString = string(parts[12]), string(parts[13])

		// In rare cases, we have embedded images with no DPI information
		// somehow...  we have to ignore these situations....
		if xdpiString == "inf" || ydpiString == "inf" {
			continue
		}

		var xdpi, _ = strconv.ParseFloat(xdpiString, 64)
		var ydpi, _ = strconv.ParseFloat(ydpiString, 64)

		if xdpi == 0 || ydpi == 0 {
			logger.Error("Invalid DPI information in line %d of %q output: %q", i+1, strings.Join(cmdParts, " "), line)
			return nil
		}

		dpis = append(dpis, pdfImageDPI{xDPI: xdpi, yDPI: ydpi})
	}

	return dpis
}
