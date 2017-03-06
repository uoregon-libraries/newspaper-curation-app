// Package fileutil holds various things this suite needs for easier
// identification and processing of files
package fileutil

import (
	"os"
	"path/filepath"
	"sort"
)

// IsDir returns true if the given path exists and is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// IsFile returns true if the given path exists and is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return info.Mode().IsRegular()
}

// Readdir wraps os.File's Readdir to handle common operations we need for
// getting a list of file info structures
func Readdir(path string) ([]os.FileInfo, error) {
	var d *os.File
	var err error

	d, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	var items []os.FileInfo
	items, err = d.Readdir(-1)
	d.Close()
	return items, err
}

// ReaddirSorted calls Readdir and sorts the results
func ReaddirSorted(path string) ([]os.FileInfo, error) {
	var fi, err = Readdir(path)
	if err == nil {
		sort.Sort(byName(fi))
	}

	return fi, err
}

// byName implements sort.Interface for sorting os.FileInfo data by name
type byName []os.FileInfo

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return n[i].Name() < n[j].Name() }

// SortFileInfos sorts a slice of os.FileInfo data by the underlying filename
func SortFileInfos(list []os.FileInfo) {
	sort.Sort(byName(list))
}

// FindDirectories returns a list of all directories or symlinks to directories
// within the given root
func FindDirectories(root string) ([]string, error) {
	var results []string
	var items, err = ReaddirSorted(root)
	if err != nil {
		// Don't fail on permission errors, just skip the dir
		if os.IsPermission(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, i := range items {
		var fName = i.Name()
		var path = filepath.Join(root, fName)
		var realPath = path
		if i.Mode()&os.ModeSymlink != 0 {
			realPath, err = os.Readlink(path)
			if err != nil {
				return nil, err
			}
			// Symlinks kind of suck - they can be absolute or relative, and if
			// they're relative we have to make them absolute....
			if !filepath.IsAbs(realPath) {
				realPath = filepath.Join(root, realPath)
			}

			i, err = os.Stat(realPath)
			if err != nil {
				// Don't fail on permission errors, just skip the file/dir
				if os.IsPermission(err) {
					continue
				}
				return nil, err
			}
		}
		realPath = filepath.Clean(realPath)

		// Skip anything we can't descend into
		if !i.IsDir() {
			continue
		}

		results = append(results, path)
	}

	return results, nil
}

// Find traverses the filesystem to the given depth, returning only the items
// that are found at that depth.  Traverses symlinks if they are directories.
// Returns the first error found if any occur.
func Find(root string, depth int) ([]string, error) {
	var paths = []string{root}
	var newPaths []string
	for depth > 0 {
		for _, p := range paths {
			var appendList, err = FindDirectories(p)
			if err != nil {
				return nil, err
			}
			newPaths = append(newPaths, appendList...)
		}
		paths = newPaths
		newPaths = nil
		depth--
	}

	return paths, nil
}
