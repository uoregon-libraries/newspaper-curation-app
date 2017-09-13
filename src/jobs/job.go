package jobs

import (
	"db"
	"fmt"
	"logger"
	"os"
	"path/filepath"
	"schema"
	"strings"
)

// Job wraps the DB job data and provides business logic for things like
// logging to the database
type Job struct {
	*db.Job
	Logger *logger.Logger
}

// IssueJob wraps the Job type to add things needed in all jobs tied to
// specific issues
type IssueJob struct {
	*Job
	Issue   *schema.Issue
	DBIssue *db.Issue
}

// jobLogWriter is our internal structure, which implements io.Writer in order
// to write to the database
type jobLogWriter struct {
	Job *Job
}

// Write implements io.Writer, splitting the logger output to produce log level
// and message strings for the database
func (jlw jobLogWriter) Write(p []byte) (n int, err error) {
	// Duplicate the output to stderr so we have something to grep in cases where
	// looking at logs is easier
	os.Stderr.Write(p)

	// Split the log message into its relevant parts
	var parts = strings.Split(string(p), " - ")
	if len(parts) < 4 {
		logger.Critical("Invalid logger message format")
		return 0, fmt.Errorf("invalid logger message format")
	}
	var level = parts[2]
	var message = parts[3]

	err = jlw.Job.WriteLog(level, message)
	if err != nil {
		logger.Critical("Unable to write log message: %s", err)
		return 0, err
	}

	return len(p), nil
}

// NewJob wraps the given db.Job and sets up a logger
func NewJob(dbj *db.Job) *Job {
	var j = &Job{Job: dbj}
	j.Logger = &logger.Logger{
		TimeFormat: "2006/01/02 15:04:05.000",
		AppName:    filepath.Base(os.Args[0]),
		Output:     jobLogWriter{j},
	}
	return j
}
