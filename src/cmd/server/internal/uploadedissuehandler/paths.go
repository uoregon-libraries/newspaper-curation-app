package uploadedissuehandler

import (
	"path"
)

// TitlePath returns the relative path to the given title's issue list page
func TitlePath(name string) string {
	return path.Join(basePath, name)
}

// IssuePath returns the relative path to the given issue's PDF list page
func IssuePath(title, issue string) string {
	return path.Join(basePath, title, issue)
}

// IssueWorkflowPath returns the path to make workflow changes to the given issue
func IssueWorkflowPath(title, issue, action string) string {
	return path.Join(basePath, title, issue, "workflow", action)
}

// FilePath returns the relative path to view a given file
func FilePath(title, issue, filename string) string {
	return path.Join(basePath, title, issue, filename)
}
