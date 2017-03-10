package httpcache

import (
	"encoding/base32"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"golang.org/x/crypto/sha3"
)

// A Request contains everything needed both to look for the requested data on
// disk and fetch it and store it
type Request struct {
	URL          string
	Filename     string
	Extension    string
	Subdirectory string
}

// NewRequest sets up a Request object for use in a Client's various
// cache-enabled functions.  The filename and extension are often not a direct
// part of the URL, or else need to be more precise than a URL provides, so we
// require them to be specified here.
func NewRequest(url, filename, extension string) *Request {
	return &Request{url, filename, extension, ""}
}

// AutoRequest uses the URL to figure out filename and extension, but requires
// a sub-directory to avoid collisions since filename from URL can be overly
// simple, lacking in context, or just not very unique.  Even so, we hash the
// URL to add a few "unique" characters to the filename.
func AutoRequest(uri, subdir string) *Request {
	var r = &Request{URL: uri, Subdirectory: subdir}
	var u, _ = url.Parse(uri)
	var urlPath = u.Path
	var base = path.Base(urlPath)

	// So we didn't actually have a base.  Now what?  Pretend it's "index.html"
	if base == "." || base == "/" || base == "" {
		base = "index.html"
	}

	// Get extension, and trim that off the base
	var ext = path.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}

	var shasum = sha3.Sum512([]byte(uri))
	var hash = base32.HexEncoding.EncodeToString(shasum[:])
	// We only keep part of the hash so filenames aren't absurd
	r.Filename = base + "-" + hash[4:8] + hash[10:14]

	// We don't keep the period in the extension
	if len(ext) > 0 {
		r.Extension = ext[1:]
	}
	return r
}

// CachePath just joins the various request data strings together to determine
// the cacheable filename for a given HTTP fetch.  If a subdirectory wasn't
// specified manually (e.g., r.Subdirectory = "foo"), the extension will be
// used to break up cached items by file type.
func (r *Request) CachePath(dataPath string) string {
	var destPath string
	if r.Subdirectory == "" {
		destPath = filepath.Join(dataPath, r.Extension)
	} else {
		destPath = filepath.Join(dataPath, r.Subdirectory)
	}

	filename := fmt.Sprintf("%s.%s", r.Filename, r.Extension)
	return filepath.Join(destPath, filename)
}
