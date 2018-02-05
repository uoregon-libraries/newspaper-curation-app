package jobs

import "db"

func queueIssueJob(t JobType, issue *db.Issue, path string) error {
	var j = &db.Job{
		Type:     string(t),
		ObjectID: issue.ID,
		Location: path,
		Status:   string(JobStatusPending),
	}
	return j.Save()
}

// QueuePageSplit creates and queues a page-splitting job with the given data
func QueuePageSplit(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypePageSplit, issue, path)
}

// QueueSFTPIssueMove creates an sftp issue move job
func QueueSFTPIssueMove(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeSFTPIssueMove, issue, path)
}

// QueueMoveIssueForDerivatives creates and queues a job to move an issue dir
// into the workflow area so a derivative job can be created
func QueueMoveIssueForDerivatives(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeMoveIssueForDerivatives, issue, path)
}

// QueueMakeDerivatives creates and queues a job to generate ALTO XML and JP2s
// for an issue
func QueueMakeDerivatives(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeMakeDerivatives, issue, path)
}

// QueueBuildMETS creates and queues a job to generate the METS XML for an
// issue that's been moved through the metadata queue
func QueueBuildMETS(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeBuildMETS, issue, path)
}
