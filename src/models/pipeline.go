package models

import (
	"fmt"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Pipeline argument names are constants to let us define arg names in a way
// that ensures we don't screw up by setting an arg and then misspelling the
// reader of said arg
const (
	JobArgWorkflowStep = "WorkflowStep"
	JobArgBatchStatus  = "BatchStatus"
	JobArgLocation     = "Location"
	JobArgSource       = "Source"
	JobArgDestination  = "Destination"
	JobArgForced       = "Forced"
	JobArgMessage      = "Message"
	JobArgExclude      = "Exclude"
)

// prepareIssueJobAdvanced is a way to get an issue job ready with the
// necessary base values, but not save it immediately, to allow for more
// advanced job semantics: specifying that the job shouldn't run immediately,
// should queue a specific job ID after completion, should set the WorkflowStep
// to a custom value rather than whatever the job would normally do, etc.
func prepareIssueJobAdvanced(t JobType, issue *Issue, args map[string]string) *Job {
	var j = NewJob(t, args)
	j.ObjectID = issue.ID
	j.ObjectType = JobObjectTypeIssue
	return j
}

// prepareBatchJobAdvanced gets a batch job ready for being used elsewhere
func prepareBatchJobAdvanced(t JobType, batch *Batch, args map[string]string) *Job {
	var j = NewJob(t, args)
	j.ObjectID = batch.ID
	j.ObjectType = JobObjectTypeBatch
	return j
}

// prepareJobJobAdvanced sets up a job to manipulate... another job.
// Jobception? I think we need one more layer to achieve it, but we're getting
// pretty close.
func prepareJobJobAdvanced(t JobType, job *Job, args map[string]string) *Job {
	var j = NewJob(t, args)
	j.ObjectID = job.ID
	j.ObjectType = JobObjectTypeJob
	return j
}

// prepareIssueActionJob sets up a job to record an internal system action tied
// to the given issue.  This is a very simple wrapper around
// prepareIssueJobAdvanced that's meant to make it a lot easier to see whan an
// action is being recorded.
func prepareIssueActionJob(issue *Issue, msg string) *Job {
	return prepareIssueJobAdvanced(JobTypeIssueAction, issue, map[string]string{JobArgMessage: msg})
}

// queueForIssue sets the issue to awaiting processing, then queues the jobs,
// all in a single DB transaction to ensure the state doesn't change if the
// jobs can't queue up
func queueForIssue(issue *Issue, jobs ...*Job) error {
	var op = dbi.DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()

	issue.WorkflowStep = schema.WSAwaitingProcessing
	var err = issue.SaveOpWithoutAction(op)
	if err != nil {
		return err
	}
	return queueSerialOp(op, jobs...)
}

// queueForBatch sets the batch status to pending, then queues the jobs, all in
// a single DB transaction to ensure the state doesn't change if the jobs can't
// queue up
func queueForBatch(batch *Batch, jobs ...*Job) error {
	var op = dbi.DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()

	batch.Status = BatchStatusPending
	var err = batch.SaveOp(op)
	if err != nil {
		return err
	}
	return queueSerialOp(op, jobs...)
}

// queueSimple queues up the given set of jobs. This must *never* be used on an
// issue- or batch-focused set of jobs, as those need to have their state set
// up by queueFor(Issue|Batch).
func queueSimple(jobs ...*Job) error {
	// Shouldn't be possible, but I'd rather not crash
	if len(jobs) == 0 {
		return nil
	}

	// Don't allow the first job to be an object-focused one. This won't protect
	// against every possible scenario, but most of the time an object-focused
	// job-set will start with the object in question, so this should prevent
	// accidental calls that should have used an object-focused function
	// (queueForX)
	if jobs[0].ObjectType == JobObjectTypeBatch || jobs[0].ObjectType == JobObjectTypeIssue {
		return fmt.Errorf("queueSimple called with object type %s", jobs[0].ObjectType)
	}

	var op = dbi.DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()
	return queueSerialOp(op, jobs...)
}

// queueSerialOp attempts to save the jobs using an existing operation (for
// when a transaction needs to wrap more than just the job queueing)
func queueSerialOp(op *magicsql.Operation, jobs ...*Job) error {
	// Iterate over jobs in reverse so we can set the prior job's next-run id
	// without saving things twice
	var lastJobID int
	for i := len(jobs) - 1; i >= 0; i-- {
		var j = jobs[i]
		j.QueueJobID = lastJobID
		if i != 0 {
			j.Status = string(JobStatusOnHold)
		}
		var err = j.SaveOp(op)
		if err != nil {
			return err
		}
		lastJobID = j.ID
	}

	return op.Err()
}
