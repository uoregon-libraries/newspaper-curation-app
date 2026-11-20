package main

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/fileutil/manifest"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
)

// newsRoot is where compose.yml mounts NCA's workflow tree (the "news"
// volume). NCA's settings only know the individual directories in it, but
// make-older and report work on the whole tree.
const newsRoot = "/mnt/news"

// copySourcesCmd replaces NCA's workflow tree with the test issues. Sources
// in sftp are copied into NCA's PDF upload dir, sometimes combining an issue's
// pages into one PDF first (to test page splitting). Sources in scans are
// copied, TIFFs and PDFs, into the scan upload dir. Upload dirs come from
// splitting the source dir's name on hyphens (see parseSourceDir).
//
// Files are copied, not hard-linked: the test sources are in a different
// mount than the workflow tree, and a copy means nothing NCA (or a recipe)
// does to the workflow can damage the sources.
//
// This is meant to be a black-box step: it works with files and runs
// commands, but it must never use NCA's own packages to do so.
type copySourcesCmd struct {
	Sources string `long:"sources" default:"test/sources" description:"directory holding the test issues' scans and sftp dirs"`
}

// Execute implements flags.Commander
func (c *copySourcesCmd) Execute([]string) error {
	if !fileutil.IsDir(c.Sources) {
		return fmt.Errorf("no test issues in %q: see test/README.md", c.Sources)
	}

	var err = emptyWorkflowDirs()
	if err == nil {
		err = copySFTPs(filepath.Join(c.Sources, "sftp"), conf.PDFUploadPath)
	}
	if err == nil {
		err = copyScans(filepath.Join(c.Sources, "scans"), conf.ScanUploadPath)
	}
	return err
}

// emptyWorkflowDirs empties every directory in NCA's workflow tree, creating
// any that are missing. The directories themselves are kept, not replaced:
// some of them are bind mounts in some containers (e.g., production batches
// in the ONI agents), and a bind mount whose directory gets replaced is left
// pointing at the deleted one.
func emptyWorkflowDirs() error {
	var dirs = []string{
		conf.PDFUploadPath, conf.ScanUploadPath, conf.PDFBackupPath, conf.PDFPageReviewPath,
		conf.WorkflowPath, conf.ErroredIssuesPath, conf.BatchOutputPath, conf.BatchProductionPath,
		conf.BatchArchivePath,
	}
	for _, dir := range dirs {
		var err = emptyDir(dir)
		if err != nil {
			return fmt.Errorf("emptying %q: %w", dir, err)
		}
	}
	return nil
}

// emptyDir removes everything inside pth, creating pth if it doesn't exist
func emptyDir(pth string) error {
	var err = os.MkdirAll(pth, 0775)
	if err != nil {
		return err
	}

	var entries []os.DirEntry
	entries, err = os.ReadDir(pth)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		err = os.RemoveAll(filepath.Join(pth, entry.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

// sourceIssue holds what we know about an issue from its source dir's name
type sourceIssue struct {
	moc     string
	lccn    string
	date    string
	edition string
}

// parseSourceDir splits a source dir's name on hyphens:
//
//   - The name must have one or two hyphens
//   - moc is only set if there are two hyphens
//   - lccn is the first part after the moc
//   - date is the first 8 characters of the last part, hyphenated the way it
//     would be in an upload directory
//   - edition is whatever follows the date
//
// Values aren't otherwise validated: invalid values need testing, too.
func parseSourceDir(name string) (*sourceIssue, error) {
	var parts = strings.Split(name, "-")
	if len(parts) != 2 && len(parts) != 3 {
		return nil, errors.New("name must contain one or two hyphens")
	}

	var i = new(sourceIssue)
	if len(parts) == 3 {
		i.moc, parts = parts[0], parts[1:]
	}
	i.lccn = parts[0]
	if len(parts[1]) < 8 {
		return nil, errors.New("date and edition must be at least 8 characters")
	}
	i.edition = parts[1][8:]

	var d = parts[1][:8]
	i.date = d[:4] + "-" + d[4:6] + "-" + d[6:]

	return i, nil
}

// sourceDirs returns the parsed issue for each directory in src. A missing
// src is fine: test data can have only one kind of issue.
func sourceDirs(src string) (dirs []string, issues []*sourceIssue, err error) {
	if !fileutil.IsDir(src) {
		logger.Warnf("No %q directory: skipping", src)
		return nil, nil, nil
	}

	var entries []os.DirEntry
	entries, err = os.ReadDir(src)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var i, err = parseSourceDir(entry.Name())
		if err != nil {
			return nil, nil, fmt.Errorf("parsing source dir %q: %w", entry.Name(), err)
		}
		dirs = append(dirs, filepath.Join(src, entry.Name()))
		issues = append(issues, i)
	}
	return dirs, issues, nil
}

// copySFTPs copies born-digital issues from src (sources/sftp) into dst, the
// PDF upload dir, as <dst>/<lccn>/<date>
func copySFTPs(src, dst string) error {
	var dirs, issues, err = sourceDirs(src)
	if err != nil {
		return err
	}

	for idx, i := range issues {
		var dir = dirs[idx]
		logger.Infof("Processing SFTP directory %q", dir)
		if i.moc != "" {
			return fmt.Errorf("source dir %q: too many hyphens", dir)
		}

		var out = filepath.Join(dst, i.lccn, i.date)
		err = freshDir(out)
		if err != nil {
			return err
		}

		// Fake randomness, so each run makes the same choices: some issues get
		// their pages combined into one PDF, to test page splitting
		if crc32.ChecksumIEEE([]byte(i.date))%2 == 0 {
			err = combinePDF(dir, out)
		} else {
			err = copyFiles(dir, out, ".pdf")
		}
		if err == nil {
			err = writeManifest(out)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// copyScans copies scanned issues from src (sources/scans) into dst, the scan
// upload dir, as <dst>/<moc>/<lccn>/<date>_<edition>
func copyScans(src, dst string) error {
	var dirs, issues, err = sourceDirs(src)
	if err != nil {
		return err
	}

	for idx, i := range issues {
		var dir = dirs[idx]
		logger.Infof("Processing scans directory %q", dir)
		if i.moc == "" {
			return fmt.Errorf("source dir %q: too few hyphens", dir)
		}

		var out = filepath.Join(dst, i.moc, i.lccn, i.date+"_"+i.edition)
		err = freshDir(out)
		if err == nil {
			err = copyFiles(dir, out, ".pdf", ".tif", ".tiff")
		}
		if err == nil {
			err = writeManifest(out)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// freshDir removes pth if it exists, then creates it
func freshDir(pth string) error {
	var err = os.RemoveAll(pth)
	if err == nil {
		err = os.MkdirAll(pth, 0775)
	}
	if err != nil {
		return fmt.Errorf("replacing %q: %w", pth, err)
	}
	return nil
}

// findFiles returns the sorted list of files in dir (not its subdirectories)
// with one of the given extensions
func findFiles(dir string, exts ...string) ([]string, error) {
	var list, err = fileutil.FindIf(dir, func(i os.FileInfo) bool {
		return slices.Contains(exts, filepath.Ext(i.Name()))
	})
	if err != nil {
		return nil, fmt.Errorf("finding %q files in %q: %w", exts, dir, err)
	}

	sort.Strings(list)
	return list, nil
}

// combinePDF uses poppler's pdfunite to combine the PDFs in src into a
// single PDF in dst
func combinePDF(src, dst string) error {
	var pdfs, err = findFiles(src, ".pdf")
	if err != nil {
		return err
	}
	if len(pdfs) == 0 {
		logger.Warnf("No PDFs in %q: skipping", src)
		return nil
	}

	var cmd = exec.Command("pdfunite", append(pdfs, filepath.Join(dst, "original.pdf"))...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.Infof("Combining %d PDFs from %q", len(pdfs), src)
	err = cmd.Run()
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line != "" {
			logger.Debugf("pdfunite out: %q", line)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
		if line != "" {
			logger.Warnf("pdfunite err: %q", line)
		}
	}
	if err != nil {
		return fmt.Errorf("combining PDFs from %q: %w", src, err)
	}
	return nil
}

// copyFiles copies every file under src with one of the given extensions
// into dst
func copyFiles(src, dst string, exts ...string) error {
	var list, err = findFiles(src, exts...)
	if err != nil {
		return err
	}

	for _, file := range list {
		var out = filepath.Join(dst, filepath.Base(file))
		logger.Debugf("Copying %q to %q", file, out)
		err = copyFile(file, out)
		if err != nil {
			return fmt.Errorf("copying %q to %q: %w", file, out, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	var in, err = os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	var info os.FileInfo
	info, err = in.Stat()
	if err != nil {
		return err
	}

	var out *os.File
	out, err = os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	var closeErr = out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func writeManifest(pth string) error {
	var m = manifest.New(pth)
	var err = m.Build()
	if err == nil {
		err = m.Write()
	}
	if err != nil {
		return fmt.Errorf("writing manifest for %q: %w", pth, err)
	}
	return nil
}

// makeOlderCmd backdates the creation date in every issue manifest in the
// workflow tree, so NCA treats the issues as having sat untouched for a while
// (see DURATION_ISSUE_CONSIDERED_DANGEROUS and _NEW in .env.example)
type makeOlderCmd struct {
	Days int `long:"days" default:"4" description:"how many days ago issues were last changed"`
}

// Execute implements flags.Commander
func (c *makeOlderCmd) Execute([]string) error {
	var y, m, d = time.Now().AddDate(0, 0, -c.Days).Date()
	var created = time.Date(y, m, d, 0, 0, 0, 0, time.Local)

	var count int
	var err = filepath.WalkDir(newsRoot, func(pth string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || e.Name() != manifest.Filename {
			return err
		}

		var dir = filepath.Dir(pth)
		var m, mErr = manifest.Open(dir)
		if mErr == nil {
			m.Created = created
			mErr = m.Write()
		}
		if mErr != nil {
			return fmt.Errorf("backdating manifest in %q: %w", dir, mErr)
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}

	logger.Infof("Backdated %d manifests to %s", count, created.Format(time.RFC3339))
	return nil
}

// renamePageReviewCmd fakes a page review pass: it renames each issue's PDFs
// to NCA's naming spec (0001.pdf, 0002.pdf, etc.), in their current sort
// order, then writes a manifest for the issue so there's no need to wait for
// NCA to notice the change
type renamePageReviewCmd struct{}

// Execute implements flags.Commander
func (c *renamePageReviewCmd) Execute([]string) error {
	var root = conf.PDFPageReviewPath
	var pdfs []string
	var err = filepath.WalkDir(root, func(pth string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() && filepath.Ext(pth) == ".pdf" {
			pdfs = append(pdfs, pth)
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("finding PDFs in %q: %w", root, err)
	}
	sort.Strings(pdfs)

	var lastDir string
	var n int
	for _, pdf := range pdfs {
		var dir = filepath.Dir(pdf)
		if dir != lastDir {
			lastDir, n = dir, 0
		}
		n++

		var newName = filepath.Join(dir, fmt.Sprintf("%04d.pdf", n))
		if newName == pdf {
			continue
		}
		if fileutil.Exists(newName) {
			return fmt.Errorf("renaming %q: %q already exists", pdf, newName)
		}
		err = os.Rename(pdf, newName)
		if err != nil {
			return fmt.Errorf("renaming %q: %w", pdf, err)
		}
	}
	logger.Infof("Renumbered %d PDFs in %q", len(pdfs), root)

	return filepath.WalkDir(root, func(pth string, e fs.DirEntry, err error) error {
		if err != nil || !e.IsDir() || pth == root {
			return err
		}
		return writeManifest(pth)
	})
}
