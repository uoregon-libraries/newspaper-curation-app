package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/uoregon-libraries/gopkg/bagit"
	"github.com/uoregon-libraries/gopkg/hasher"
)

// tagmanifestFilename is the BagIt tag manifest file at the root of every
// archived bag. Its contents are SHA-256 sums of the bag's tag files, which
// themselves include the payload manifest — so a hash of this one file is a
// transitive fingerprint of the entire bag.
const tagmanifestFilename = "tagmanifest-sha256.txt"

// stRdOnly is the Linux statfs(2) mount flag indicating a read-only mount.
// The stdlib syscall package doesn't export a named constant for this, but
// the value is stable (see <linux/statfs.h>).
const stRdOnly = 0x1

// tagmanifestFingerprint returns a SHA-256 hex digest of the bag's
// tagmanifest-sha256.txt. If the file is missing or unreadable, the error is
// returned.
func tagmanifestFingerprint(bagPath string) (string, error) {
	var data, err = os.ReadFile(filepath.Join(bagPath, tagmanifestFilename))
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", tagmanifestFilename, err)
	}
	var sum = sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// validateArchive runs a full BagIt validation on the given bag. It returns
// the list of discrepancies found (empty if the bag is valid) and any I/O
// error encountered while attempting to validate.
func validateArchive(bagPath string) (discrepancies []string, err error) {
	var b = bagit.New(bagPath, hasher.NewSHA256())
	return b.Validate()
}

// archiveDirReadOnly reports whether dir's filesystem is mounted read-only.
// It uses statfs(2) to check the actual mount flags rather than probing with
// a write, because a failed write attempt can't distinguish between a
// read-only filesystem and a simple lack of write permission. Only a
// kernel-level read-only mount gives the safety guarantee we want here.
func archiveDirReadOnly(dir string) (bool, error) {
	var info, err = os.Stat(dir)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory", dir)
	}

	var stat syscall.Statfs_t
	err = syscall.Statfs(dir, &stat)
	if err != nil {
		return false, fmt.Errorf("statfs %s: %w", dir, err)
	}
	return stat.Flags&stRdOnly != 0, nil
}

// countIssuesInBatchXML counts the <issue> elements in the bag's
// data/batch.xml. Namespaces in the batch XML are ignored; matching is by
// local element name.
func countIssuesInBatchXML(bagPath string) (int, error) {
	var path = filepath.Join(bagPath, "data", "batch.xml")
	var data, err = os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc struct {
		Issues []struct{} `xml:"issue"`
	}
	err = xml.Unmarshal(data, &doc)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", path, err)
	}
	return len(doc.Issues), nil
}
