// Package fileutil holds various things this suite needs for easier
// identification and processing of files
package fileutil

import (
	"os"
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
