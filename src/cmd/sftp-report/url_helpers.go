package main

import (
	"path"
)

// URL paths
const (
	HomePath = ""
)

// FullPath uses the webroot, if not empty, to join together all the path parts
// with a slash, returning an absolute path to something
func FullPath(parts ...string) string {
	if Webroot != "" {
		parts = append([]string{Webroot}, parts...)
	}
	return path.Join(parts...)
}
