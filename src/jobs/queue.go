package jobs

import (
	"path/filepath"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// These constants let us define arg names in a way that ensures we don't screw
// up by setting an arg and then misspelling the reader of said arg
const (
	wsArg   = "WorkflowStep"
	bsArg   = "BatchStatus"
	locArg  = "Location"
	srcArg  = "Source"
	destArg = "Destination"
)

// PrepareJobAdvanced gets a job of any kind set up with sensible defaults
func PrepareJobAdvanced(t models.JobType, args map[string]string) *models.Job {
	return models.NewJob(t, args)
}

// PrepareIssueJobAdvanced is a way to get an issue job ready with the
// necessary base values, but not save it immediately, to allow for more
// advanced job semantics: specifying that the job shouldn't run immediately,
// should queue a specific job ID after completion, should set the WorkflowStep
// to a custom value rather than whatever the job would normally do, etc.
func PrepareIssueJobAdvanced(t models.JobType, issue *models.Issue, args map[string]string) *models.Job {
	var j = PrepareJobAdvanced(t, args)
	j.ObjectID = issue.ID
	j.ObjectType = models.JobObjectTypeIssue
	return j
}

// PrepareBatchJobAdvanced gets a batch job ready for being used elsewhere
func PrepareBatchJobAdvanced(t models.JobType, batch *models.Batch, args map[string]string) *models.Job {
	var j = PrepareJobAdvanced(t, args)
	j.ObjectID = batch.ID
	j.ObjectType = models.JobObjectTypeBatch
	return j
}

// QueueSerial attempts to save the jobs (in a transaction), setting the first
// one as ready to run while the others become effectively dependent on the
// prior job in the list
func QueueSerial(jobs ...*models.Job) error {
	var op = dbi.DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()

	// Iterate over jobs in reverse so we can set the prior job's next-run id
	// without saving things twice
	var lastJobID int
	for i := len(jobs) - 1; i >= 0; i-- {
		var j = jobs[i]
		j.QueueJobID = lastJobID
		if i != 0 {
			j.Status = string(models.JobStatusOnHold)
		}
		var err = j.SaveOp(op)
		if err != nil {
			return err
		}
		lastJobID = j.ID
	}

	return op.Err()
}

func makeWSArgs(ws schema.WorkflowStep) map[string]string {
	return map[string]string{wsArg: string(ws)}
}

func makeBSArgs(bs string) map[string]string {
	return map[string]string{bsArg: string(bs)}
}

func makeLocArgs(loc string) map[string]string {
	return map[string]string{locArg: loc}
}

func makeSrcDstArgs(src, dest string) map[string]string {
	return map[string]string{
		srcArg:  src,
		destArg: dest,
	}
}

// QueueSFTPIssueMove queues up an issue move into the workflow area followed
// by a page-split and then a move to the page review area
func QueueSFTPIssueMove(issue *models.Issue, c *config.Config) error {
	var workflowDir = filepath.Join(c.WorkflowPath, issue.HumanName)
	var workflowWIPDir = filepath.Join(c.WorkflowPath, ".wip-"+issue.HumanName)
	var pageReviewDir = filepath.Join(c.PDFPageReviewPath, issue.HumanName)
	var pageReviewWIPDir = filepath.Join(c.PDFPageReviewPath, ".wip-"+issue.HumanName)
	var backupLoc = filepath.Join(c.PDFBackupPath, issue.HumanName)

	return QueueSerial(
		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSAwaitingProcessing)),

		// Move the issue to the workflow location
		PrepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, workflowWIPDir)),
		PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		PrepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(workflowDir)),

		// Clean dotfiles and then kick off the page splitter
		PrepareJobAdvanced(models.JobTypeCleanFiles, makeLocArgs(workflowDir)),
		PrepareIssueJobAdvanced(models.JobTypePageSplit, issue, makeLocArgs(workflowWIPDir)),

		// This gets a bit weird.  What's in the issue location dir is the original
		// upload, which we back up since we may need to reprocess the PDFs from
		// these originals.  Once we've backed up (syncdir + killdir), we move the
		// WIP files back into the proper workflow folder...  which is then
		// promptly moved out to the page review area.
		PrepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(workflowDir, backupLoc)),
		PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(workflowDir)),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueBackupLoc, issue, makeLocArgs(backupLoc)),
		PrepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),

		// Now we move the issue data to the page review area for manual
		// processing, again in multiple idempotent steps
		PrepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(workflowDir, pageReviewWIPDir)),
		PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(workflowDir)),
		PrepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(pageReviewWIPDir, pageReviewDir)),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(pageReviewDir)),

		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSAwaitingPageReview)),
	)
}

// QueueMoveIssueForDerivatives creates jobs to move issues into the workflow
// and then immediately generate derivatives
func QueueMoveIssueForDerivatives(issue *models.Issue, workflowPath string) error {
	var workflowDir = filepath.Join(workflowPath, issue.HumanName)
	var workflowWIPDir = filepath.Join(workflowPath, ".wip-"+issue.HumanName)

	return QueueSerial(
		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSAwaitingProcessing)),

		PrepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, workflowWIPDir)),
		PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		PrepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(workflowDir)),

		PrepareJobAdvanced(models.JobTypeCleanFiles, makeLocArgs(workflowDir)),
		PrepareIssueJobAdvanced(models.JobTypeMakeDerivatives, issue, nil),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSReadyForMetadataEntry)),
	)
}

// QueueFinalizeIssue creates and queues jobs that get an issue ready for
// batching.  Currently this means generating the METS XML file and copying
// archived PDFs (if born-digital) into the issue directory.
func QueueFinalizeIssue(issue *models.Issue) error {
	// Some jobs aren't queued up unless there's a backup, so we actually
	// generate a list of jobs programatically instead of inline
	var jobs []*models.Job
	jobs = append(jobs, PrepareIssueJobAdvanced(models.JobTypeBuildMETS, issue, nil))

	if issue.BackupLocation != "" {
		jobs = append(jobs, PrepareIssueJobAdvanced(models.JobTypeArchiveBackups, issue, nil))
		jobs = append(jobs, PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)))
		jobs = append(jobs, PrepareIssueJobAdvanced(models.JobTypeSetIssueBackupLoc, issue, makeLocArgs("")))
	}

	jobs = append(jobs, PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSReadyForBatching)))

	return QueueSerial(jobs...)
}

// QueueMakeBatch sets up the jobs for generating a batch on disk: generating
// the directories and hard-links, making the batch XML, putting the batch
// where it can be loaded onto staging, and generating the bagit manifest.
// Nothing can happen automatically after all this until the batch is verified
// on staging.
func QueueMakeBatch(batch *models.Batch, batchOutputPath string) error {
	var wipDir = filepath.Join(batchOutputPath, ".wip-"+batch.FullName())
	var finalDir = filepath.Join(batchOutputPath, batch.FullName())
	return QueueSerial(
		PrepareBatchJobAdvanced(models.JobTypeCreateBatchStructure, batch, makeLocArgs(wipDir)),
		PrepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs(wipDir)),
		PrepareBatchJobAdvanced(models.JobTypeMakeBatchXML, batch, nil),
		PrepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		PrepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs(finalDir)),
		PrepareBatchJobAdvanced(models.JobTypeSetBatchStatus, batch, makeBSArgs(models.BatchStatusQCReady)),
		PrepareBatchJobAdvanced(models.JobTypeWriteBagitManifest, batch, nil),
	)
}
