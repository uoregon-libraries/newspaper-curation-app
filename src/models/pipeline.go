package models

import (
	"fmt"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// PipelineName is a simple string meant to ensure a controlled list of
// pipelines for consistency and easier filtering. All pipelines must have a
// valid name.
type PipelineName string

// Valid pipeline names
const (
	PNSFTPIssueMove           PipelineName = "SFTPIssueMove"
	PNMoveIssueForDerivatives PipelineName = "MoveIssueForDerivatives"
	PNQueueIssueForReview     PipelineName = "QueueIssueForReview"
	PNFinalizeIssue           PipelineName = "FinalizeIssue"
	PNMakeBatch               PipelineName = "MakeBatch"
	PNRemoveErroredIssue      PipelineName = "RemoveErroredIssue"
	PNPurgeStuckIssue         PipelineName = "PurgeStuckIssue"
	PNFinalizeIssueFlagging   PipelineName = "FinalizeIssueFlagging"
	PNBatchDeletion           PipelineName = "BatchDeletion"
	PNGoLiveProcess           PipelineName = "GoLiveProcess"
)

// A Pipeline is a connected series of independent jobs which all perform tasks
// for a single purpose. Each job is given a numeric "sequence" number, where
// the lower the value, the higher the priority. e.g., no job may run until all
// jobs with a lower sequence value have completed successfully.
//
// In complex Pipelines, some jobs might share a sequence, meaning they could
// be run in parallel. We don't plan to implement that in the job runner, but
// it is still a signal that those jobs are independent of one another.
//
// In even more complex Pipelines, a job may spawn another job meant to run
// before whatever would have come next. This just means a "sub-job" that has
// the same sequence as its creator, ensuring whatever would run next in the
// sequence has to wait for the new job to run.
type Pipeline struct {
	ID          int64 `sql:",primary"`
	Name        string
	Description string
	ObjectType  string
	ObjectID    int64
	CreatedAt   time.Time `sql:",readonly"`
	StartedAt   time.Time
	CompletedAt time.Time

	jobs []*Job
}

// newPipeline creates a pipeline with the given description. Pipelines should
// generally not be created outside this package as they are meant to be
// created only when queueing up a bunch of jobs.
func newPipeline(name PipelineName, desc string) *Pipeline {
	return &Pipeline{Name: string(name), Description: desc}
}

// findPipelines returns all Pipeline instances that match the filter
func findPipelines(where string, args ...any) ([]*Pipeline, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var list []*Pipeline
	op.Select("pipelines", &Pipeline{}).Where(where, args...).AllObjects(&list)
	return list, op.Err()
}

// findPipeline pulls the pipeline object for the given id
func findPipeline(id int64) (*Pipeline, error) {
	var list, err = findPipelines("id = ?", id)
	if len(list) == 0 {
		return nil, err
	}
	return list[0], err
}

// QueueIssueJobs sets the issue to awaiting processing, removes it from any
// "desk" it's currently on, then queues the jobs, all in a single DB
// transaction to ensure the state doesn't change if the jobs can't queue up
func QueueIssueJobs(name PipelineName, issue *Issue, jobs ...*Job) error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	issue.WorkflowOwnerID = 0
	issue.WorkflowStep = schema.WSAwaitingProcessing
	var err = issue.SaveOpWithoutAction(op)
	if err != nil {
		return err
	}

	var p = newPipeline(name, fmt.Sprintf("issue %s", issue.Key()))
	p.ObjectType = JobObjectTypeIssue
	p.ObjectID = issue.ID
	return p.queueSerialOp(op, jobs...)
}

// QueueBatchJobs sets the batch status to pending, then queues the jobs, all
// in a single DB transaction to ensure the state doesn't change if the jobs
// can't queue up
func QueueBatchJobs(name PipelineName, batch *Batch, jobs ...*Job) error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	batch.Status = BatchStatusPending
	var err = batch.SaveOpWithoutAction(op)
	if err != nil {
		return err
	}

	var p = newPipeline(name, fmt.Sprintf("batch %s", batch.FullName()))
	p.ObjectType = JobObjectTypeBatch
	p.ObjectID = batch.ID
	return p.queueSerialOp(op, jobs...)
}

// QueueJobs queues up the given set of jobs. This must *never* be used on an
// issue- or batch-focused set of jobs, as those need to have their state set
// up by Queue(Issue|Batch)Jobs.
func QueueJobs(name PipelineName, description string, jobs ...*Job) error {
	// Shouldn't be possible, but I'd rather not crash
	if len(jobs) == 0 {
		return nil
	}

	// Don't allow the first job to be an object-focused one. This won't protect
	// against every possible scenario, but most of the time an object-focused
	// job-set will start with the object in question, so this should prevent
	// accidental calls that should have used an object-focused function
	// (queueXJobs)
	if jobs[0].ObjectType == JobObjectTypeBatch || jobs[0].ObjectType == JobObjectTypeIssue {
		return fmt.Errorf("QueueJobs called with object type %s", jobs[0].ObjectType)
	}

	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	var p = newPipeline(name, description)
	return p.queueSerialOp(op, jobs...)
}

// saveOp uses an existing DB operation to save the Pipeline. This is private
// to avoid use of Pipelines outside of very strictly-controlled situations.
func (p *Pipeline) saveOp(op *magicsql.Operation) error {
	op.Save("pipelines", p)
	return op.Err()
}

// queueSerialOp attempts to save the jobs using an existing operation (for
// when a transaction needs to wrap more than just the job queueing)
func (p *Pipeline) queueSerialOp(op *magicsql.Operation, jobs ...*Job) error {
	// Start by saving the pipeline itself so we have an ID for the jobs
	var err = p.saveOp(op)
	if err != nil {
		return fmt.Errorf("save pipeline %#v: %s", p, err)
	}

	// For now, we just add jobs and give them a sequence based on where they
	// appear in the list. This will need to change to allow complex pipelines to
	// manually set up job sequences.
	for i, job := range jobs {
		job.PipelineID = p.ID
		job.Sequence = i + 1
		if i == 0 {
			job.Status = string(JobStatusPending)
		} else {
			job.Status = string(JobStatusOnHold)
		}
		var err = job.SaveOp(op)
		if err != nil {
			return fmt.Errorf("save job %#v: %s", job, err)
		}
	}

	return op.Err()
}

// Jobs returns all jobs associated with the given pipeline.
//
// Results are cached after the first successful query, so a new Pipeline
// should be read from the database to forcibly re-read jobs. This should
// almost never be necessary.
func (p *Pipeline) Jobs() ([]*Job, error) {
	if p.jobs != nil {
		return p.jobs, nil
	}

	return findJobs("pipeline_id = ?", p.ID)
}
