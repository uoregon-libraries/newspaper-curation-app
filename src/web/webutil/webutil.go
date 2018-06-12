// Package webutil holds functions and data that other packages may need in
// order to generate URLs, find static files, etc.
package webutil

import (
	"fmt"
	"html/template"
	"net/url"
	"path"
	"strings"
)

// Webroot must be set by main to tell us where we are within the main website,
// such as "/reports", and is used to generate absolute paths to various
// handlers and site assets
var Webroot string

// WorkflowPath is the path to the workflow directory for serving IIIF images
var WorkflowPath string

// IIIFBaseURL is the IIIF server URL
var IIIFBaseURL string

// FullPath uses the webroot, if not empty, to join together all the path parts
// with a slash, returning an absolute path to something
func FullPath(parts ...string) string {
	parts = append([]string{Webroot}, parts...)
	if Webroot == "" {
		parts[0] = "/"
	} else if Webroot[0] != '/' {
		parts[0] = "/" + parts[0]
	}
	return path.Join(parts...)
}

// StaticPath returns the absolute path to static assets (CSS, JS, etc)
func StaticPath(dir, file string) string {
	return FullPath("static", dir, file)
}

// HomePath returns the absolute path to the home page (title list)
func HomePath() string {
	return FullPath("")
}

// ImageURL takes a file and constructs an absolute web path string
func ImageURL(file string) string {
	return StaticPath("images", file)
}

// IncludeCSS generates a <link> tag with an absolute path for including the
// given file's CSS.  ".css" is automatically appended to the filename for less
// verbose use.
func IncludeCSS(file string) template.HTML {
	var path = StaticPath("css", file+".css")
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" type="text/css" href="%s" />`, path))
}

// RawCSS generates a <link> tag with an absolute path for including the
// given file's CSS.  It doesn't assume the path is /css, and it doesn't
// auto-append ".css".
func RawCSS(file string) template.HTML {
	var path = StaticPath("", file)
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" type="text/css" href="%s" />`, path))
}

// IncludeJS generates a <script> tag with an absolute path for including the
// given file's JS.  ".js" is automatically appended to the filename for less
// verbose use.
func IncludeJS(file string) template.HTML {
	var path = StaticPath("js", file+".js")
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, path))
}

// RawJS generates a <script> tag with an absolute path for including the given
// file's JS.  It doesn't assume the path is /js, and it doesn't auto-append
// ".js".
func RawJS(file string) template.HTML {
	var path = StaticPath("", file)
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, path))
}

// IIIFInfoURL returns what a IIIF viewer needs to find a JP2
func IIIFInfoURL(jp2Path string) string {
	var relPath = strings.Replace(jp2Path, WorkflowPath+"/", "", 1)
	relPath = path.Clean(relPath)
	var identifier = url.PathEscape(relPath)
	return fmt.Sprintf("%s/%s", IIIFBaseURL, path.Join(identifier, "info.json"))
}
