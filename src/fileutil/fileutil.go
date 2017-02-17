// Package fileutil holds various things this suite needs for easier
// identification and processing of files
package fileutil

import "os"

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
