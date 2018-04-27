package issuefinderhandler

import (
	"path"
)

// SearchPath returns the relative path to a search for a given issue key string
func SearchPath(key string) string {
	return path.Join(basePath, key)
}
