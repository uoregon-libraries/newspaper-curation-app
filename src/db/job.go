package db

import "time"

// JobLog is a single log entry attached to a job
type JobLog struct {
	ID        int `sql:",primary"`
	JobID     int
	CreatedAt time.Time `sql:",readonly"`
	LogLevel  string
	Message   string
}

// A Job is anything the app needs to process and track in the background
type Job struct {
	ID            int       `sql:",primary"`
	CreatedAt     time.Time `sql:",readonly"`
	NextAttemptAt time.Time `sql:",noinsert"`
	Type          string    `sql:"job_type"`
	ObjectID      int
	Location      string
	Status        string
	logs          []*JobLog
}

// FindJob gets a job by its id
func FindJob(id int) (*Job, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var j = &Job{}
	var ok = op.Select("jobs", &Job{}).Where("id = ?", id).First(j)
	if !ok {
		return nil, op.Err()
	}
	return j, op.Err()
}

// findJobs wraps all the job finding functionality so helpers can be
// one-liners.  This is purposely *not* exported to enforce a stricter API.
func findJobs(where string, args ...interface{}) ([]*Job, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*Job
	op.Select("jobs", &Job{}).Where(where, args...).AllObjects(&list)
	return list, op.Err()
}

// FindJobsByStatusAndType returns all jobs of the given status and type
func FindJobsByStatusAndType(st string, t string) ([]*Job, error) {
	return findJobs("status = ? AND job_type = ?", st, t)
}

// FindJobsForIssueID returns all jobs tied to the given issue
func FindJobsForIssueID(id int) ([]*Job, error) {
	return findJobs("object_id = ?", id)
}

// Logs lazy-loads all logs for this job from the database
func (j *Job) Logs() []*JobLog {
	if j.logs == nil {
		var op = DB.Operation()
		op.Dbg = Debug
		op.Select("job_logs", &JobLog{}).Where("job_id = ?", j.ID).AllObjects(j.logs)
	}

	return j.logs
}

// WriteLog stores a log message on this job
func (j *Job) WriteLog(level string, message string) error {
	var l = &JobLog{JobID: j.ID, LogLevel: level, Message: message}
	var op = DB.Operation()
	op.Dbg = Debug
	op.Save("job_logs", l)
	return op.Err()
}

// Save creates or updates the Job in the jobs table
func (j *Job) Save() error {
	var op = DB.Operation()
	op.Dbg = Debug
	op.Save("jobs", j)
	return op.Err()
}
