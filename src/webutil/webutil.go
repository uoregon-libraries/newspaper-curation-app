// Package webutil holds functions and data that other packages may need in
// order to generate URLs, find static files, etc.
package webutil

import (
	"fmt"
	"html/template"
	"path"
)

// Webroot must be set by main to tell us where we are within the main website,
// such as "/reports", and is used to generate absolute paths to various
// handlers and site assets
var Webroot string

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

// ImageURL takes a file and constructs an absolute web path string
func ImageURL(file string) string {
	return FullPath("images", file)
}

// IncludeCSS generates a <link> tag with an absolute path for including the
// given file's CSS.  ".css" is automatically appended to the filename for less
// verbose use.
func IncludeCSS(file string) template.HTML {
	var path = FullPath("css", file+".css")
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" type="text/css" href="%s" />`, path))
}

// RawCSS generates a <link> tag with an absolute path for including the
// given file's CSS.  It doesn't assume the path is /css, and it doesn't
// auto-append ".css".
func RawCSS(file string) template.HTML {
	var path = FullPath(file)
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" type="text/css" href="%s" />`, path))
}

// IncludeJS generates a <script> tag with an absolute path for including the
// given file's JS.  ".js" is automatically appended to the filename for less
// verbose use.
func IncludeJS(file string) template.HTML {
	var path = FullPath("js", file+".js")
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, path))
}

// RawJS generates a <script> tag with an absolute path for including the given
// file's JS.  It doesn't assume the path is /js, and it doesn't auto-append
// ".js".
func RawJS(file string) template.HTML {
	var path = FullPath(file)
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, path))
}
