package jobs

import (
	"fmt"
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
	wsArg      = "WorkflowStep"
	bsArg      = "BatchStatus"
	locArg     = "Location"
	srcArg     = "Source"
	destArg    = "Destination"
	forcedArg  = "Forced"
	msgArg     = "Message"
	excludeArg = "Exclude"
)

// prepareJobAdvanced gets a job of any kind set up with sensible defaults
func prepareJobAdvanced(t models.JobType, args map[string]string) *models.Job {
	return models.NewJob(t, args)
}

// prepareIssueJobAdvanced is a way to get an issue job ready with the
// necessary base values, but not save it immediately, to allow for more
// advanced job semantics: specifying that the job shouldn't run immediately,
// should queue a specific job ID after completion, should set the WorkflowStep
// to a custom value rather than whatever the job would normally do, etc.
func prepareIssueJobAdvanced(t models.JobType, issue *models.Issue, args map[string]string) *models.Job {
	var j = prepareJobAdvanced(t, args)
	j.ObjectID = issue.ID
	j.ObjectType = models.JobObjectTypeIssue
	return j
}

// prepareBatchJobAdvanced gets a batch job ready for being used elsewhere
func prepareBatchJobAdvanced(t models.JobType, batch *models.Batch, args map[string]string) *models.Job {
	var j = prepareJobAdvanced(t, args)
	j.ObjectID = batch.ID
	j.ObjectType = models.JobObjectTypeBatch
	return j
}

// prepareJobJobAdvanced sets up a job to manipulate... another job.
// Jobception? I think we need one more layer to achieve it, but we're getting
// pretty close.
func prepareJobJobAdvanced(t models.JobType, job *models.Job, args map[string]string) *models.Job {
	var j = prepareJobAdvanced(t, args)
	j.ObjectID = job.ID
	j.ObjectType = models.JobObjectTypeJob
	return j
}

// prepareIssueActionJob sets up a job to record an internal system action tied
// to the given issue.  This is a very simple wrapper around
// prepareIssueJobAdvanced that's meant to make it a lot easier to see whan an
// action is being recorded.
func prepareIssueActionJob(issue *models.Issue, msg string) *models.Job {
	return prepareIssueJobAdvanced(models.JobTypeIssueAction, issue, map[string]string{msgArg: msg})
}

// queueForIssue sets the issue to awaiting processing, then queues the jobs,
// all in a single DB transaction to ensure the state doesn't change if the
// jobs can't queue up
func queueForIssue(issue *models.Issue, jobs ...*models.Job) error {
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
func queueForBatch(batch *models.Batch, jobs ...*models.Job) error {
	var op = dbi.DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()

	batch.Status = models.BatchStatusPending
	var err = batch.SaveOp(op)
	if err != nil {
		return err
	}
	return queueSerialOp(op, jobs...)
}

// queueSimple queues up the given set of jobs. This must *never* be used on an
// issue- or batch-focused set of jobs, as those need to have their state set
// up by queueFor(Issue|Batch).
func queueSimple(jobs ...*models.Job) error {
	// Shouldn't be possible, but I'd rather not crash
	if len(jobs) == 0 {
		return nil
	}

	// Don't allow the first job to be an object-focused one. This won't protect
	// against every possible scenario, but most of the time an object-focused
	// job-set will start with the object in question, so this should prevent
	// accidental calls that should have used an object-focused function
	// (queueForX)
	if jobs[0].ObjectType == models.JobObjectTypeBatch || jobs[0].ObjectType == models.JobObjectTypeIssue {
		return fmt.Errorf("queueSimple called with object type %s", jobs[0].ObjectType)
	}

	var op = dbi.DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()
	return queueSerialOp(op, jobs...)
}

// queueSerialOp attempts to save the jobs using an existing operation (for
// when a transaction needs to wrap more than just the job queueing)
func queueSerialOp(op *magicsql.Operation, jobs ...*models.Job) error {
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

	return queueForIssue(issue,
		// Move the issue to the workflow location
		prepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, workflowWIPDir)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		prepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),
		prepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(workflowDir)),

		// Clean dotfiles and then kick off the page splitter
		prepareJobAdvanced(models.JobTypeCleanFiles, makeLocArgs(workflowDir)),
		prepareIssueJobAdvanced(models.JobTypePageSplit, issue, makeLocArgs(workflowWIPDir)),

		// This gets a bit weird.  What's in the issue location dir is the original
		// upload, which we back up since we may need to reprocess the PDFs from
		// these originals.  Once we've backed up (syncdir + killdir), we move the
		// WIP files back into the proper workflow folder...  which is then
		// promptly moved out to the page review area.
		prepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(workflowDir, backupLoc)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(workflowDir)),
		prepareIssueJobAdvanced(models.JobTypeSetIssueBackupLoc, issue, makeLocArgs(backupLoc)),
		prepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),

		// Now we move the issue data to the page review area for manual
		// processing, again in multiple idempotent steps
		prepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(workflowDir, pageReviewWIPDir)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(workflowDir)),
		prepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(pageReviewWIPDir, pageReviewDir)),
		prepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(pageReviewDir)),

		prepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSAwaitingPageReview)),
		prepareIssueActionJob(issue, "Moved issue from SFTP into NCA"),
	)
}

// QueueMoveIssueForDerivatives creates jobs to move issues into the workflow,
// make all issues' pages numbered nicely, and then generate derivatives
func QueueMoveIssueForDerivatives(issue *models.Issue, workflowPath string) error {
	var workflowDir = filepath.Join(workflowPath, issue.HumanName)
	var workflowWIPDir = filepath.Join(workflowPath, ".wip-"+issue.HumanName)

	return queueForIssue(issue,
		prepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, workflowWIPDir)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		prepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),
		prepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(workflowDir)),

		prepareJobAdvanced(models.JobTypeCleanFiles, makeLocArgs(workflowDir)),
		prepareIssueJobAdvanced(models.JobTypeRenumberPages, issue, nil),
		prepareIssueJobAdvanced(models.JobTypeMakeDerivatives, issue, nil),
		prepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSReadyForMetadataEntry)),
		prepareIssueActionJob(issue, "Created issue derivatives"),
	)
}

// QueueForceDerivatives will forcibly regenerate all derivatives for an issue.
// During the processing, the issue's workflow step is set to "awaiting
// processing", and only gets set back to its previous value on successful
// completion of the other jobs.
func QueueForceDerivatives(issue *models.Issue) error {
	var currentStep = issue.WorkflowStep
	return queueForIssue(issue,
		prepareIssueJobAdvanced(models.JobTypeMakeDerivatives, issue, makeForcedArgs()),
		prepareIssueJobAdvanced(models.JobTypeBuildMETS, issue, makeForcedArgs()),
		prepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(currentStep)),
		prepareIssueActionJob(issue, "Force-regenerated issue derivatives"),
	)
}

// QueueFinalizeIssue creates and queues jobs that get an issue ready for
// batching.  Currently this means generating the METS XML file and copying
// archived PDFs (if born-digital) into the issue directory.
func QueueFinalizeIssue(issue *models.Issue) error {
	// Some jobs aren't queued up unless there's a backup, so we actually
	// generate a list of jobs programatically instead of inline
	var jobs []*models.Job
	jobs = append(jobs, prepareIssueJobAdvanced(models.JobTypeBuildMETS, issue, nil))

	if issue.BackupLocation != "" {
		jobs = append(jobs, prepareIssueJobAdvanced(models.JobTypeArchiveBackups, issue, nil))
		jobs = append(jobs, prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)))
		jobs = append(jobs, prepareIssueJobAdvanced(models.JobTypeSetIssueBackupLoc, issue, makeLocArgs("")))
	}

	jobs = append(jobs, prepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSReadyForBatching)))
	jobs = append(jobs, prepareIssueActionJob(issue, "Issue prepped for batching"))

	return queueForIssue(issue, jobs...)
}

// QueueMakeBatch sets up the jobs for generating a batch on disk: generating
// the directories and hard-links, making the batch XML, putting the batch
// where it can be loaded onto staging, and generating the bagit manifest.
// Nothing can happen automatically after all this until the batch is verified
// on staging.
func QueueMakeBatch(batch *models.Batch, batchOutputPath string) error {
	return queueForBatch(batch, getJobsForMakeBatch(batch, batchOutputPath)...)
}

// getJobsForMakeBatch returns all jobs needed to generate a batch. This is needed
// by two different higher-level tasks.
func getJobsForMakeBatch(batch *models.Batch, pth string) []*models.Job {
	var wipDir = filepath.Join(pth, ".wip-"+batch.FullName())
	var finalDir = filepath.Join(pth, batch.FullName())
	return []*models.Job{
		prepareBatchJobAdvanced(models.JobTypeCreateBatchStructure, batch, makeLocArgs(wipDir)),
		prepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs(wipDir)),
		prepareBatchJobAdvanced(models.JobTypeMakeBatchXML, batch, nil),
		prepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		prepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs(finalDir)),
		prepareBatchJobAdvanced(models.JobTypeSetBatchStatus, batch, makeBSArgs(models.BatchStatusStagingReady)),
		prepareBatchJobAdvanced(models.JobTypeWriteBagitManifest, batch, nil),
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
	var jobs = getJobsForRemoveErroredIssue(issue, erroredIssueRoot)
	return queueForIssue(issue, jobs...)
}

// QueuePurgeStuckIssue builds jobs for removing an issue that had critical
// failures on one or more jobs. Any waiting (on-hold) jobs still tied to the
// issue are removed, as are failed jobs, and then the issue is purged with
// data a dev can use to look into the problem more closely.
func QueuePurgeStuckIssue(issue *models.Issue, erroredIssueRoot string) error {
	var allJobs, err = models.FindJobsForIssueID(issue.ID)
	if err != nil {
		return err
	}

	var purgeReason = "Issue failed getting through workflow:\n"
	var jobs []*models.Job
	for _, j := range allJobs {
		switch models.JobStatus(j.Status) {
		case models.JobStatusFailed, models.JobStatusOnHold:
			if j.Status == string(models.JobStatusFailed) {
				purgeReason += fmt.Sprintf("- Job %d (%s) failed too many times\n", j.ID, j.Type)
			}
			var jj = prepareJobJobAdvanced(models.JobTypeCancelJob, j, nil)
			jobs = append(jobs, jj)
		}
	}
	jobs = append(jobs, prepareIssueActionJob(issue, purgeReason))
	jobs = append(jobs, getJobsForRemoveErroredIssue(issue, erroredIssueRoot)...)

	return queueSimple(jobs...)
}

// getJobsForRemoveErroredIssue returns the list of jobs for removing the given
// errored issue, suitable for use in a queue* call
func getJobsForRemoveErroredIssue(issue *models.Issue, erroredIssueRoot string) []*models.Job {
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
		prepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, contentDir)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		prepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs(contentDir)),
		prepareIssueJobAdvanced(models.JobTypeMoveDerivatives, issue, makeLocArgs(derivDir)),
		prepareIssueJobAdvanced(models.JobTypeWriteActionLog, issue, nil),
	)

	// If we have a backup, archive it and remove its files
	if issue.BackupLocation != "" {
		jobs = append(jobs,
			prepareIssueJobAdvanced(models.JobTypeArchiveBackups, issue, nil),
			prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)),
			prepareIssueJobAdvanced(models.JobTypeSetIssueBackupLoc, issue, makeLocArgs("")),
		)
	}

	// Move to the final location and update metadata
	jobs = append(jobs,
		prepareJobAdvanced(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		prepareIssueJobAdvanced(models.JobTypeIgnoreIssue, issue, nil),
		prepareIssueJobAdvanced(models.JobTypeSetIssueLocation, issue, makeLocArgs("")),
		prepareIssueJobAdvanced(models.JobTypeSetIssueWS, issue, makeWSArgs(schema.WSUnfixableMetadataError)),
		prepareIssueActionJob(issue, "Errored issue removed from NCA"),
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
		prepareBatchJobAdvanced(models.JobTypeSetBatchStatus, batch, makeBSArgs(models.BatchStatusPending)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		prepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs("")),
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
			prepareIssueJobAdvanced(models.JobTypeRemoveFile, i.Issue, makeLocArgs(i.Issue.METSFile())),
			prepareIssueJobAdvanced(models.JobTypeFinalizeBatchFlaggedIssue, i.Issue, nil),
		)
	}

	// Remove all the no-longer-useful flagged issue data
	jobs = append(jobs, prepareBatchJobAdvanced(models.JobTypeEmptyBatchFlaggedIssuesList, batch, nil))

	// Regenerate batch
	jobs = append(jobs, getJobsForMakeBatch(batch, batchOutputPath)...)

	return queueForBatch(batch, jobs...)
}

// QueueBatchForDeletion is used when all issues in a batch need to be
// rejected, rendering the batch unnecessary (and useless).
func QueueBatchForDeletion(batch *models.Batch, flagged []*models.FlaggedIssue) error {
	// This is essentially a copy of the finalization job list, except there's no
	// regenerate-batch step
	var jobs []*models.Job

	// Destroy batch dir
	jobs = append(jobs,
		prepareBatchJobAdvanced(models.JobTypeSetBatchStatus, batch, makeBSArgs(models.BatchStatusPending)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		prepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs("")),
	)

	// Finalize flagged issues
	for _, i := range flagged {
		jobs = append(jobs,
			prepareIssueJobAdvanced(models.JobTypeRemoveFile, i.Issue, makeLocArgs(i.Issue.METSFile())),
			prepareIssueJobAdvanced(models.JobTypeFinalizeBatchFlaggedIssue, i.Issue, nil),
		)
	}

	// Remove all the no-longer-useful flagged issue data
	jobs = append(jobs, prepareBatchJobAdvanced(models.JobTypeEmptyBatchFlaggedIssuesList, batch, nil))

	// Destroy the batch
	jobs = append(jobs, prepareBatchJobAdvanced(models.JobTypeDeleteBatch, batch, nil))

	return queueForBatch(batch, jobs...)
}

// QueueCopyBatchForProduction sets the given batch to pending, then queues up
// the necessary jobs to get it ready for a production load
func QueueCopyBatchForProduction(batch *models.Batch, prodBatchRoot string) error {
	// We need batch status *and* staging purge flag to be updated instantly, so
	// we start a tx here and queue manually instead of using queueForBatch.
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	batch.Status = models.BatchStatusPending
	batch.NeedStagingPurge = true
	var err = batch.SaveOp(op)
	if err != nil {
		return err
	}

	// Our sync job is special - it requires us to have exclusions, so we're just
	// building a custom args list
	var args = makeSrcDstArgs(batch.Location, filepath.Join(prodBatchRoot, batch.FullName()))
	args[excludeArg] = `*.tif,*.tiff,*.TIF,*.TIFF,*.tar.bz,*.tar`

	return queueSerialOp(op,
		prepareBatchJobAdvanced(models.JobTypeValidateTagManifest, batch, nil),
		prepareJobAdvanced(models.JobTypeSyncDir, args),
		prepareBatchJobAdvanced(models.JobTypeSetBatchStatus, batch, makeBSArgs(models.BatchStatusPassedQC)),
	)
}

// QueueBatchGoLiveProcess fires off all jobs needed to call a batch live and
// ready for archiving. These jobs should only be queued up after a batch has
// been ingested into the production ONI instance.
func QueueBatchGoLiveProcess(batch *models.Batch, batchArchivePath string) error {
	var finalPath = filepath.Join(batchArchivePath, batch.FullName())
	return queueForBatch(batch,
		prepareJobAdvanced(models.JobTypeSyncDir, makeSrcDstArgs(batch.Location, finalPath)),
		prepareJobAdvanced(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		prepareBatchJobAdvanced(models.JobTypeSetBatchLocation, batch, makeLocArgs("")),
		prepareBatchJobAdvanced(models.JobTypeMarkBatchLive, batch, nil),
	)
}
