package jobs

import "db"

// QueuePageSplit creates and queues a page-splitting job with the given data
func QueuePageSplit(t JobType, id int, path string) error {
	var j = &db.Job{Type: string(t), ObjectID: id, Location: path, Status: string(JobStatusPending)}
	return j.Save()
}
