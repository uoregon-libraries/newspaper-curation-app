package jobs

import (
	"path/filepath"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// These constants let us define arg names in a way that ensures we don't screw
// up by setting an arg and then misspelling the reader of said arg
const (
	wsArg     = "WorkflowStep"
	bsArg     = "BatchStatus"
	locArg    = "Location"
	srcArg    = "Source"
	destArg   = "Destination"
	forcedArg = "Forced"
	msgArg    = "Message"
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

// PrepareIssueActionJob sets up a job to record an internal system action tied
// to the given issue.  This is a very simple wrapper around
// PrepareIssueJobAdvanced that's meant to make it a lot easier to see whan an
// action is being recorded.
func PrepareIssueActionJob(issue *models.Issue, msg string) *models.Job {
	return PrepareIssueJobAdvanced(models.JobTypeIssueAction, issue, map[string]string{msgArg: msg})
}

// QueueSerial attempts to save the jobs (in a transaction), setting the first
// one as ready to run while the others become effectively dependent on the
// prior job in the list
func QueueSerial(jobs ...*models.Job) error {
	var op = dbi.DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()
	return QueueSerialOp(op, jobs...)
}

// QueueSerialOp attempts to save the jobs using an existing operation (for
// when a transaction needs to wrap more than just the job queueing), but is
// otherwise the same as QueueSerial.
func QueueSerialOp(op *magicsql.Operation, jobs ...*models.Job) error {
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

func makeForcedArgs() map[string]string {
	return map[string]string{forcedArg: forcedArg}
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
		PrepareIssueActionJob(issue, "Moved issue from SFTP into NCA"),
	)
}

// QueueMoveIssueForDerivatives creates jobs to move issues into the workflow,
// make all issues' pages numbered nicely, and then generate derivatives
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
		PrepareIssueJobAdvanced(models.JobTypeRenumberPages, issue, nil),
		PrepareIssueJobAdvanced(models.JobTypeMakeDerivatives, issue, nil),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSReadyForMetadataEntry)),
		PrepareIssueActionJob(issue, "Created issue derivatives"),
	)
}

// QueueForceDerivatives will forcibly regenerate all derivatives for an issue.
// During the processing, the issue's workflow step is set to "awaiting
// processing", and only gets set back to its previous value on successful
// completion of the other jobs.
func QueueForceDerivatives(issue *models.Issue) error {
	var currentStep = issue.WorkflowStep
	return QueueSerial(
		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSAwaitingProcessing)),
		PrepareIssueJobAdvanced(models.JobTypeMakeDerivatives, issue, makeForcedArgs()),
		PrepareIssueJobAdvanced(models.JobTypeBuildMETS, issue, makeForcedArgs()),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(currentStep)),
		PrepareIssueActionJob(issue, "Force-regenerated issue derivatives"),
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
	jobs = append(jobs, PrepareIssueActionJob(issue, "Issue prepped for batching"))

	return QueueSerial(jobs...)
}

// QueueMakeBatch sets up the jobs for generating a batch on disk: generating
// the directories and hard-links, making the batch XML, putting the batch
// where it can be loaded onto staging, and generating the bagit manifest.
// Nothing can happen automatically after all this until the batch is verified
// on staging.
func QueueMakeBatch(batch *models.Batch, batchOutputPath string) error {
	return QueueSerial(getJobsForMakeBatch(batch, batchOutputPath)...)
}

// getJobsForMakeBatch returns all jobs needed to generate a batch. This is needed
// by two different higher-level tasks.
func getJobsForMakeBatch(batch *models.Batch, pth string) []*models.Job {
	var wipDir = filepath.Join(pth, ".wip-"+batch.FullName())
	var finalDir = filepath.Join(pth, batch.FullName())
	return []*models.Job{
		PrepareBatchJobAdvanced(models.JobTypeCreateBatchStructure, batch, makeLocArgs(wipDir)),
		PrepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs(wipDir)),
		PrepareBatchJobAdvanced(models.JobTypeMakeBatchXML, batch, nil),
		PrepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		PrepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs(finalDir)),
		PrepareBatchJobAdvanced(models.JobTypeSetBatchStatus, batch, makeBSArgs(models.BatchStatusStagingReady)),
		PrepareBatchJobAdvanced(models.JobTypeWriteBagitManifest, batch, nil),
	}
}

// QueueRemoveErroredIssue builds jobs necessary to take an issue permanently
// out of NCA's workflow:
//
// - The issue is flagged in the database as no longer being in NCA
// - The issue directory is copied to the error location and then the original is removed
// - The original uploads, if relevant, are moved into the error directory
// - The derivatives are put under a sibling sub-dir from the primary files
func QueueRemoveErroredIssue(issue *models.Issue, erroredIssueRoot string) error {
	var jobs = GetJobsForRemoveErroredIssue(issue, erroredIssueRoot)
	return QueueSerial(jobs...)
}

// GetJobsForRemoveErroredIssue returns the list of jobs for removing the given
// errored issue, suitable for use in a QueueSerial or QueueSerialOp call
func GetJobsForRemoveErroredIssue(issue *models.Issue, erroredIssueRoot string) []*models.Job {
	var dt = time.Now()
	var dateSubdir = dt.Format("2006-01")
	var rootDir = filepath.Join(erroredIssueRoot, dateSubdir)
	var wipDir = filepath.Join(rootDir, ".wip-"+issue.HumanName)
	var finalDir = filepath.Join(rootDir, issue.HumanName)
	var contentDir = filepath.Join(wipDir, "content")
	var derivDir = filepath.Join(wipDir, "derivatives")

	// This is another set of jobs that has conditional steps, so we build it up
	var jobs []*models.Job

	// The first steps are unconditional: move the issue to the WIP location,
	// move derivative images to the correct subdir so the wip/content dir
	// consists solely of primary files, and write out the action log file.
	jobs = append(jobs,
		PrepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, contentDir)),
		PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(contentDir)),
		PrepareIssueJobAdvanced(models.JobTypeMoveDerivatives, issue, makeLocArgs(derivDir)),
		PrepareIssueJobAdvanced(models.JobTypeWriteActionLog, issue, nil),
	)

	// If we have a backup, archive it and remove its files
	if issue.BackupLocation != "" {
		jobs = append(jobs,
			PrepareIssueJobAdvanced(models.JobTypeArchiveBackups, issue, nil),
			PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)),
			PrepareIssueJobAdvanced(models.JobTypeSetIssueBackupLoc, issue, makeLocArgs("")),
		)
	}

	// Move to the final location and update metadata
	jobs = append(jobs,
		PrepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		PrepareIssueJobAdvanced(models.JobTypeIgnoreIssue, issue, nil),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs("")),
		PrepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSUnfixableMetadataError)),
		PrepareIssueActionJob(issue, "Errored issue removed from NCA"),
	)

	return jobs
}

// QueueBatchFinalizeIssueFlagging generates jobs for removing flagged issues
// from a batch which failed QC, then rebuilding the batch
func QueueBatchFinalizeIssueFlagging(batch *models.Batch, flagged []*models.FlaggedIssue, batchOutputPath string) error {
	// This is yet another set of jobs that has steps we build out rather than
	// just having a hard-coded list queued up
	var jobs []*models.Job

	// Destroy batch dir jobs - note that the batch dir contains hard links and
	// easily rebuilt metadata (e.g., the bagit info), so this is not truly a
	// destructive operation
	jobs = append(jobs,
		PrepareBatchJobAdvanced(models.JobTypeSetBatchStatus, batch, makeBSArgs(models.BatchStatusPending)),
		PrepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		PrepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs("")),
	)

	// Remove issues one at a time so we can easily resume / restart. Removing an
	// issue means we first remove the METS XML file, and only when that succeeds
	// do we do that database side. Filesystem jobs can fail in totally stupid
	// ways (NFS mount dropping) and we want those to retry separately from the
	// rest of the job.
	//
	// Note that in a perfect world each issue job could actually be running
	// concurrently, but the job runner doesn't have the capability for one job
	// to be dependent on a group of jobs. We'd prefer to keep the issues
	// separate jobs and take that small performance hit rather than trying to
	// add that level of complexity to job processing.
	for _, i := range flagged {
		jobs = append(jobs,
			PrepareIssueJobAdvanced(models.JobTypeRemoveFile, i.Issue, makeLocArgs(i.Issue.METSFile())),
			PrepareIssueJobAdvanced(models.JobTypeFinalizeBatchFlaggedIssue, i.Issue, nil),
		)
	}

	// Remove all the no-longer-useful flagged issue data
	jobs = append(jobs, PrepareBatchJobAdvanced(models.JobTypeEmptyBatchFlaggedIssuesList, batch, nil))

	// Regenerate batch
	jobs = append(jobs, getJobsForMakeBatch(batch, batchOutputPath)...)

	return QueueSerial(jobs...)
}
