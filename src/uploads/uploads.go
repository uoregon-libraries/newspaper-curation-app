// Package uploads is for the one-off validation / queue processing which only
// applies to issues which aren't yet in the workflow.  e.g., verifying that
// scanned issues have PDFs and TIFFs for each page, reporting
// pre-process-specific errors when trying to bring an issue into NCA's
// workflow, etc.
//
// The package is meant to be usable from the web as well as command-line tasks
// as a way to ensure consistency when processing uploaded issues.
package uploads
