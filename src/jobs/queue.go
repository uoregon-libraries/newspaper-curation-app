package jobs

import "db"

// QueuePageSplit creates and queues a page-splitting job with the given data
func QueuePageSplit(issue *db.Issue, path string) error {
	var j = &db.Job{
		Type:     string(JobTypePageSplit),
		ObjectID: issue.ID,
		Location: path,
		Status:   string(JobStatusPending),
	}
	return j.Save()
}
