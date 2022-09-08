// copy-sources.go hard-links source issues' files into the fakemount to test
// various aspects of processing.  We hard-link PDFs in sources/sftp into
// fakemount/sftp, sometimes combining the pages into a new PDF first (to test
// page splitting), other times just copying them as-is. SFTP dir names are
// translated to match what we use in our seed data.  We then hard-link TIFF
// and PDF files in sources/scans into fakemount/scans.  Directories for SFTP
// publisher are taken by splitting the sources dir on hyphen, and the same
// happens for scans, but we also expect an org code.
//
// This is meant to be a 100% black-box test.  It can interact with the data
// and filesystem, run commands, and maybe even force data into the database,
// but it should never be allowed to use any of the local packages like "jobs"
// in any direct way.
//
// Eventually this should be part of a bigger test suite which lives in its own
// test-only docker-compose setup, runs various jobs, and tests output.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/fileutil/manifest"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

var l = logger.New(logger.Debug, false)

func wrap(msg string) {
	fmt.Fprint(os.Stderr, wordutils.Wrap(msg, 80))
	fmt.Fprintln(os.Stderr)
}

func usageFail(format string, args ...interface{}) {
	wrap(fmt.Sprintf(format, args...))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "usage: go run copy-sources.go <test directory>")
	os.Exit(1)
}

func main() {
	if len(os.Args) != 2 {
		usageFail("You must specify exactly one argument")
	}
	var testDir, err = filepath.Abs(os.Args[1])
	if err != nil || fileutil.IsDir(testDir) == false {
		usageFail("You must specify a valid path to test")
	}

	refreshFakemount(testDir)
	moveSFTPs(testDir)
	moveScans(testDir)
}

// issue holds a simplified version of an issue's data - basically just the
// stuff we can pull from the parsed directory name
type issue struct {
	moc     string
	lccn    string
	date    string
	edition string
}

// sftpdir returns the sftp dirname based on our known seed data. This is
// hard-coded and really bad since we don't provide seed data, but it is a
// necessary evil to move this test suite forward(ish) for the SFTPGo feature.
func (i *issue) sftpdir() string {
	switch i.lccn {
	case "2004260523":
		return "appealtribune"
	case "sn00063621":
		return "keizertimes"
	case "sn83008376":
		return "astorian"
	case "sn96088087":
		return "polkitemizer"
	case "sn99063854":
		return "vernoniaeagle"
	}

	//  If we don't have a match we just return the LCCN as-is rather than try to
	//  look anything up. Long-term, though, we really should just connect to the
	//  NCA database for this whole process.
	return i.lccn
}

// getDirParts splits dirname on hyphens, and translates the data such that:
// - an error is returned if dirname had neither 2 nor 3 hyphens
// - moc is only set if dirname had 3 hyphens
// - lccn is set to the first part after the moc
// - date is converted from the first 8 characters of the last part, and
//   hyphens are added so it's formatted like it would be when in an sftp/scan
//   upload location
// - edition is parsed from the final two characters
func getDirParts(dirName string) (*issue, error) {
	var parts = strings.Split(dirName, "-")
	if len(parts) != 2 && len(parts) != 3 {
		return nil, errors.New("dirname must contain two or three hyphens")
	}

	// We do *not* validate data, as invalid values need testing, too
	var i = new(issue)
	if len(parts) == 3 {
		i.moc, parts = parts[0], parts[1:]
	}
	i.lccn = parts[0]
	i.edition = parts[1][8:]

	// Convert the date to a hyphenated form
	var dateString = parts[1][:8]
	i.date = dateString[:4] + "-" + dateString[4:6] + "-" + dateString[6:]

	return i, nil
}

func refreshFakemount(testDir string) {
	for _, dir := range []string{"backup/originals", "outgoing", "page-review", "scans", "sftp", "workflow", "errors"} {
		var fullPath = filepath.Join(testDir, "fakemount", dir)
		if err := os.RemoveAll(fullPath); err != nil {
			l.Criticalf("Unable to delete %q: %s", fullPath, err)
			os.Exit(255)
		}
		if err := os.MkdirAll(fullPath, 0775); err != nil {
			l.Criticalf("Unable to create %q: %s", fullPath, err)
			os.Exit(255)
		}
	}
}

func moveSFTPs(testDir string) {
	var sftpSourcePath = filepath.Join(testDir, "sources", "sftp")
	var sftpDestPath = filepath.Join(testDir, "fakemount", "sftp")
	var infos, err = ioutil.ReadDir(sftpSourcePath)
	if err != nil {
		l.Fatalf("Unable to read ./sources/sftp: %s", err)
	}
	for _, info := range infos {
		var dirName = info.Name()
		l.Infof("Processing SFTP directory %q", info.Name())
		var issue, err = getDirParts(dirName)
		if err != nil {
			l.Fatalf("Unable to parse directory %q: %s", dirName, err)
		}
		if issue.moc != "" {
			l.Fatalf("Unable to parse directory %q: too many hyphens", dirName)
		}

		var outPath = filepath.Join(sftpDestPath, issue.sftpdir(), issue.date)
		err = os.RemoveAll(outPath)
		if err != nil {
			l.Fatalf("Unable to clear SFTP destination directory %q: %s", outPath, err)
		}

		err = os.MkdirAll(outPath, 0775)
		if err != nil {
			l.Fatalf("Unable to create SFTP destination directory %q: %s", outPath, err)
		}

		// Now use fake-randomness to decide if we're building a combined pdf
		var hashval = crc32.ChecksumIEEE([]byte(issue.date))
		var issueSrcPath = filepath.Join(sftpSourcePath, info.Name())
		if hashval%2 == 0 {
			combinePDF(issueSrcPath, outPath)
		} else {
			linkFiles(issueSrcPath, outPath, ".pdf")
		}

		makeManifest(outPath)
	}
}

func getFiles(dir string, exts ...string) ([]string, error) {
	l.Debugf("getFiles(%q, %q)", dir, exts)
	var fileList, err = fileutil.FindIf(dir, func(i os.FileInfo) bool {
		l.Debugf("Scanning %q", i.Name())
		for _, ext := range exts {
			if filepath.Ext(i.Name()) == ext {
				return true
			}
		}
		return false
	})

	sort.Strings(fileList)
	return fileList, err
}

// combinePDF relies on poppler utilities to combine PDFs in the source
// directory to create a single PDF in the destination directory
func combinePDF(src, dst string) {
	// Find all PDF files
	var pdfs, err = getFiles(src, ".pdf")
	if err != nil {
		l.Fatalf("Unable to scan for PDF files in %q: %s", src, err)
	}

	var args = append(pdfs, filepath.Join(dst, "original.pdf"))
	var cmd = exec.Command("pdfunite", args...)
	var stdout = new(bytes.Buffer)
	var stderr = new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	l.Infof("Running %q %q to generate a combined PDF", cmd.Path, cmd.Args)
	err = cmd.Run()

	if len(stdout.Bytes()) > 0 {
		for _, line := range strings.Split(string(stdout.Bytes()), "\n") {
			l.Debugf("pdfunite out: %q", line)
		}
	}

	if len(stderr.Bytes()) > 0 {
		for _, line := range strings.Split(string(stderr.Bytes()), "\n") {
			l.Warnf("pdfunite err: %q", line)
		}
	}
	if err != nil {
		l.Fatalf("Unable to create a combined PDF: %s", err)
	}
}

// linkFiles finds all files in src that have one of the extensions given, and
// hard-links them in dst
func linkFiles(src, dst string, extensions ...string) {
	// Find all files
	var fileList, err = getFiles(src, extensions...)
	if err != nil {
		l.Fatalf("Unable to scan for %q files in %q: %s", extensions, src, err)
	}

	for _, file := range fileList {
		var fileBase = filepath.Base(file)
		var outFile = filepath.Join(dst, fileBase)
		l.Debugf("Hard-linking %q to %q", file, outFile)
		os.Link(file, outFile)
	}
}

func moveScans(testDir string) {
	var scansSourcePath = filepath.Join(testDir, "sources", "scans")
	var scansDestPath = filepath.Join(testDir, "fakemount", "scans")
	var infos, err = ioutil.ReadDir(scansSourcePath)
	if err != nil {
		l.Fatalf("Unable to read %q: %s", scansSourcePath, err)
	}
	for _, info := range infos {
		var dirName = info.Name()
		l.Infof("Processing scans directory %q", info.Name())
		var issue, err = getDirParts(dirName)
		if err != nil {
			l.Fatalf("Unable to parse directory %q: %s", dirName, err)
		}
		if issue.moc == "" {
			l.Fatalf("Unable to parse directory %q: too few hyphens", dirName)
		}

		var outPath = filepath.Join(scansDestPath, issue.moc, issue.lccn, issue.date+"_"+issue.edition)
		err = os.RemoveAll(outPath)
		if err != nil {
			l.Fatalf("Unable to clear scans destination directory %q: %s", outPath, err)
		}

		err = os.MkdirAll(outPath, 0775)
		if err != nil {
			l.Fatalf("Unable to create scans destination directory %q: %s", outPath, err)
		}

		var issueSrcPath = filepath.Join(scansSourcePath, info.Name())
		linkFiles(issueSrcPath, outPath, ".pdf", ".tif", ".tiff")

		makeManifest(outPath)
	}
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
