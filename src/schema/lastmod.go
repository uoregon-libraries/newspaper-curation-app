package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

const manifestFilename = ".manifest"

type fileInfo struct {
	Path     string
	Size     int64
	Checksum string
}

func newFileInfo(loc string, e os.DirEntry) (fileInfo, error) {
	var fd = fileInfo{Path: filepath.Join(loc, e.Name())}
	var info, err = e.Info()
	if err != nil {
		return fd, fmt.Errorf("reading info for %q: %w", fd.Path, err)
	}

	fd.Size = info.Size()
	fd.Checksum, err = fileutil.CRC32(fd.Path)
	if err != nil {
		return fd, fmt.Errorf("crc32 for %q: %w", fd.Path, err)
	}

	return fd, nil
}

type manifest struct {
	Path    string
	Created time.Time
	Files   []fileInfo
}

func newManifest(location string) *manifest {
	return &manifest{Path: location, Created: time.Now()}
}

func (m *manifest) build() error {
	var entries, err = os.ReadDir(m.Path)
	if err != nil {
		return fmt.Errorf("reading dir %q: %s", m.Path, err)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return fmt.Errorf("reading dir %q: one or more entries are not a regular file", m.Path)
		}

		// Skip the manifest as well as any hidden files - these have no bearing
		// once issues move to NCA. We explicitly check for the manifest in case we
		// change the constant string to not be hidden for some reason.
		if entry.Name()[0] == '.' || entry.Name() == manifestFilename {
			continue
		}

		var fd, err = newFileInfo(m.Path, entry)
		if err != nil {
			return fmt.Errorf("reading dir %q: %s", m.Path, err)
		}
		m.Files = append(m.Files, fd)
	}
	return nil
}

func (m *manifest) read() error {
	var data, err = ioutil.ReadFile(m.filename())
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	return nil
}

func (m *manifest) filename() string {
	return filepath.Join(m.Path, manifestFilename)
}

func (m *manifest) write() error {
	var data, err = json.Marshal(m)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(m.filename(), data, 0600)
}

func (m *manifest) sortFiles() {
	sort.Slice(m.Files, func(i, j int) bool {
		return m.Files[i].Path < m.Files[j].Path
	})
}

func (m *manifest) equiv(m2 *manifest) bool {
	if len(m.Files) != len(m2.Files) {
		return false
	}
	m.sortFiles()
	m2.sortFiles()

	for i := range m.Files {
		if m.Files[i] != m2.Files[i] {
			return false
		}
	}

	return true
}

// LastModified tells us when *any* change happened in an issue's folder.  This
// will return a meaningless value on live issues.
func (i *Issue) LastModified() time.Time {
	if i.WorkflowStep == WSInProduction {
		return time.Time{}
	}

	// Set up the two manifest structures for this issue
	var m1, m2 = newManifest(i.Location), newManifest(i.Location)

	// First build a manifest of everything in the issue dir
	var err = m1.build()
	if err != nil {
		// This can happen when an issue is being moved by another process while
		// this process is scanning it, so we just return time.Now() and ignore the
		// (likely invalid) error
		if errors.Is(err, fs.ErrNotExist) {
			return time.Now()
		}

		logger.Errorf("Unable to read dir %q: %s", i.Location, err)
		return time.Now()
	}

	// Second, read the existing manifest
	err = m2.read()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Errorf("Unable to read existing manifest for issue in %q: %s", i.Location, err)
		return time.Now()
	}

	// Different existing manifest (including not having an existing manifest)?
	// Write new data and return the current time.
	if !m1.equiv(m2) {
		err = m1.write()
		if err != nil {
			logger.Errorf("Unable to write new manifest for issue in %q: %s", i.Location, err)
		}
		return m1.Created
	}

	// Manifests are the same? Return the existing manifest's creation time.
	return m2.Created
}
