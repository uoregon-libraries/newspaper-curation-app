package db

// JobLog is a single log entry attached to a job
type JobLog struct {
	ID       int `sql:",primary"`
	JobID    int
	LogLevel string
	Message  string
}

// A Job is anything the app needs to process and track in the background
type Job struct {
	ID       int    `sql:",primary"`
	Type     string `sql:"job_type"`
	ObjectID int
	Location string
	Status   string
	logs     []*JobLog
}

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

// Logs lazy-loads all logs for this job from the database
func (j *Job) Logs() []*JobLog {
	if j.logs == nil {
		var op = DB.Operation()
		op.Dbg = Debug
		op.Select("job_logs", &JobLog{}).Where("job_id = ?", j.ID).AllObjects(j.logs)
	}

	return j.logs
}
