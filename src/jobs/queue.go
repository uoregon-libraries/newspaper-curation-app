package jobs

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

func makeWSArgs(ws schema.WorkflowStep) map[string]string {
	return map[string]string{models.JobArgWorkflowStep: string(ws)}
}

func makeBSArgs(bs string) map[string]string {
	return map[string]string{models.JobArgBatchStatus: string(bs)}
}

func makeLocArgs(loc string) map[string]string {
	return map[string]string{models.JobArgLocation: loc}
}

func makeForcedArgs() map[string]string {
	return map[string]string{models.JobArgForced: models.JobArgForced}
}

func makeSrcDstArgs(src, dest string) map[string]string {
	return map[string]string{
		models.JobArgSource:      src,
		models.JobArgDestination: dest,
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

	return models.QueueIssueJobs(issue,
		// Move the issue to the workflow location
		models.NewJob(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, workflowWIPDir)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),
		issue.Job(models.JobTypeSetIssueLocation, makeLocArgs(workflowDir)),

		// Clean dotfiles and then kick off the page splitter
		models.NewJob(models.JobTypeCleanFiles, makeLocArgs(workflowDir)),
		issue.Job(models.JobTypePageSplit, makeLocArgs(workflowWIPDir)),

		// This gets a bit weird.  What's in the issue location dir is the original
		// upload, which we back up since we may need to reprocess the PDFs from
		// these originals.  Once we've backed up (syncdir + killdir), we move the
		// WIP files back into the proper workflow folder...  which is then
		// promptly moved out to the page review area.
		models.NewJob(models.JobTypeSyncDir, makeSrcDstArgs(workflowDir, backupLoc)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(workflowDir)),
		issue.Job(models.JobTypeSetIssueBackupLoc, makeLocArgs(backupLoc)),
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),

		// Now we move the issue data to the page review area for manual
		// processing, again in multiple idempotent steps
		models.NewJob(models.JobTypeSyncDir, makeSrcDstArgs(workflowDir, pageReviewWIPDir)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(workflowDir)),
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(pageReviewWIPDir, pageReviewDir)),
		issue.Job(models.JobTypeSetIssueLocation, makeLocArgs(pageReviewDir)),

		issue.Job(models.JobTypeSetIssueWS, makeWSArgs(schema.WSAwaitingPageReview)),
		issue.ActionJob("Moved issue from SFTP into NCA"),
	)
}

// QueueMoveIssueForDerivatives creates jobs to move issues into the workflow,
// make all issues' pages numbered nicely, and then generate derivatives
func QueueMoveIssueForDerivatives(issue *models.Issue, workflowPath string) error {
	var workflowDir = filepath.Join(workflowPath, issue.HumanName)
	var workflowWIPDir = filepath.Join(workflowPath, ".wip-"+issue.HumanName)

	return models.QueueIssueJobs(issue,
		models.NewJob(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, workflowWIPDir)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(workflowWIPDir, workflowDir)),
		issue.Job(models.JobTypeSetIssueLocation, makeLocArgs(workflowDir)),

		models.NewJob(models.JobTypeCleanFiles, makeLocArgs(workflowDir)),
		issue.Job(models.JobTypeRenumberPages, nil),
		issue.Job(models.JobTypeMakeDerivatives, nil),
		issue.Job(models.JobTypeSetIssueWS, makeWSArgs(schema.WSReadyForMetadataEntry)),
		issue.ActionJob("Created issue derivatives"),
	)
}

// QueueForceDerivatives will forcibly regenerate all derivatives for an issue.
// During the processing, the issue's workflow step is set to "awaiting
// processing", and only gets set back to its previous value on successful
// completion of the other jobs.
func QueueForceDerivatives(issue *models.Issue) error {
	var currentStep = issue.WorkflowStep
	return models.QueueIssueJobs(issue,
		issue.Job(models.JobTypeMakeDerivatives, makeForcedArgs()),
		issue.Job(models.JobTypeBuildMETS, makeForcedArgs()),
		issue.Job(models.JobTypeSetIssueWS, makeWSArgs(currentStep)),
		issue.ActionJob("Force-regenerated issue derivatives"),
	)
}

// QueueFinalizeIssue creates and queues jobs that get an issue ready for
// batching.  Currently this means generating the METS XML file and copying
// archived PDFs (if born-digital) into the issue directory.
func QueueFinalizeIssue(issue *models.Issue) error {
	// Some jobs aren't queued up unless there's a backup, so we actually
	// generate a list of jobs programatically instead of inline
	var jobs []*models.Job
	jobs = append(jobs, issue.Job(models.JobTypeBuildMETS, nil))

	if issue.BackupLocation != "" {
		jobs = append(jobs, issue.Job(models.JobTypeArchiveBackups, nil))
		jobs = append(jobs, models.NewJob(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)))
		jobs = append(jobs, issue.Job(models.JobTypeSetIssueBackupLoc, makeLocArgs("")))
	}

	jobs = append(jobs, issue.Job(models.JobTypeSetIssueWS, makeWSArgs(schema.WSReadyForBatching)))
	jobs = append(jobs, issue.ActionJob("Issue prepped for batching"))

	return models.QueueIssueJobs(issue, jobs...)
}

// QueueMakeBatch sets up the jobs for generating a batch on disk: generating
// the directories and hard-links, making the batch XML, putting the batch
// where it can be loaded onto staging, and generating the bagit manifest.
// Nothing can happen automatically after all this until the batch is verified
// on staging.
func QueueMakeBatch(batch *models.Batch, batchOutputPath string) error {
	return models.QueueBatchJobs(batch, getJobsForMakeBatch(batch, batchOutputPath)...)
}

// getJobsForMakeBatch returns all jobs needed to generate a batch. This is needed
// by two different higher-level tasks.
func getJobsForMakeBatch(batch *models.Batch, pth string) []*models.Job {
	var wipDir = filepath.Join(pth, ".wip-"+batch.FullName())
	var finalDir = filepath.Join(pth, batch.FullName())
	return []*models.Job{
		batch.Job(models.JobTypeCreateBatchStructure, makeLocArgs(wipDir)),
		batch.Job(models.JobTypeSetBatchLocation, makeLocArgs(wipDir)),
		batch.Job(models.JobTypeMakeBatchXML, nil),
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		batch.Job(models.JobTypeSetBatchLocation, makeLocArgs(finalDir)),
		batch.Job(models.JobTypeSetBatchStatus, makeBSArgs(models.BatchStatusStagingReady)),
		batch.Job(models.JobTypeWriteBagitManifest, nil),
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
	return models.QueueIssueJobs(issue, jobs...)
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
			var jj = j.Job(models.JobTypeCancelJob, nil)
			jobs = append(jobs, jj)
		}
	}
	jobs = append(jobs, issue.ActionJob(purgeReason))
	jobs = append(jobs, getJobsForRemoveErroredIssue(issue, erroredIssueRoot)...)

	return models.QueueJobs(jobs...)
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
		models.NewJob(models.JobTypeSyncDir, makeSrcDstArgs(issue.Location, contentDir)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(issue.Location)),
		issue.Job(models.JobTypeSetIssueLocation, makeLocArgs(contentDir)),
		issue.Job(models.JobTypeMoveDerivatives, makeLocArgs(derivDir)),
		issue.Job(models.JobTypeWriteActionLog, nil),
	)

	// If we have a backup, archive it and remove its files
	if issue.BackupLocation != "" {
		jobs = append(jobs,
			issue.Job(models.JobTypeArchiveBackups, nil),
			models.NewJob(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)),
			issue.Job(models.JobTypeSetIssueBackupLoc, makeLocArgs("")),
		)
	}

	// Move to the final location and update metadata
	jobs = append(jobs,
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		issue.Job(models.JobTypeIgnoreIssue, nil),
		issue.Job(models.JobTypeSetIssueLocation, makeLocArgs("")),
		issue.Job(models.JobTypeSetIssueWS, makeWSArgs(schema.WSUnfixableMetadataError)),
		issue.ActionJob("Errored issue removed from NCA"),
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
		batch.Job(models.JobTypeSetBatchStatus, makeBSArgs(models.BatchStatusPending)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		batch.Job(models.JobTypeSetBatchLocation, makeLocArgs("")),
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
			i.Issue.Job(models.JobTypeRemoveFile, makeLocArgs(i.Issue.METSFile())),
			i.Issue.Job(models.JobTypeFinalizeBatchFlaggedIssue, nil),
		)
	}

	// Remove all the no-longer-useful flagged issue data
	jobs = append(jobs, batch.Job(models.JobTypeEmptyBatchFlaggedIssuesList, nil))

	// Regenerate batch
	jobs = append(jobs, getJobsForMakeBatch(batch, batchOutputPath)...)

	return models.QueueBatchJobs(batch, jobs...)
}

// QueueBatchForDeletion is used when all issues in a batch need to be
// rejected, rendering the batch unnecessary (and useless).
func QueueBatchForDeletion(batch *models.Batch, flagged []*models.FlaggedIssue) error {
	// This is essentially a copy of the finalization job list, except there's no
	// regenerate-batch step
	var jobs []*models.Job

	// Destroy batch dir
	jobs = append(jobs,
		batch.Job(models.JobTypeSetBatchStatus, makeBSArgs(models.BatchStatusPending)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		batch.Job(models.JobTypeSetBatchLocation, makeLocArgs("")),
	)

	// Finalize flagged issues
	for _, i := range flagged {
		jobs = append(jobs,
			i.Issue.Job(models.JobTypeRemoveFile, makeLocArgs(i.Issue.METSFile())),
			i.Issue.Job(models.JobTypeFinalizeBatchFlaggedIssue, nil),
		)
	}

	// Remove all the no-longer-useful flagged issue data
	jobs = append(jobs, batch.Job(models.JobTypeEmptyBatchFlaggedIssuesList, nil))

	// Destroy the batch
	jobs = append(jobs, batch.Job(models.JobTypeDeleteBatch, nil))

	return models.QueueBatchJobs(batch, jobs...)
}

// QueueCopyBatchForProduction sets the given batch to pending, then queues up
// the necessary jobs to get it ready for a production load
func QueueCopyBatchForProduction(batch *models.Batch, prodBatchRoot string) error {
	// Our sync job is special - it requires us to have exclusions, so we're just
	// building a custom args list
	var args = makeSrcDstArgs(batch.Location, filepath.Join(prodBatchRoot, batch.FullName()))
	args[models.JobArgExclude] = `*.tif,*.tiff,*.TIF,*.TIFF,*.tar.bz,*.tar`

	// TODO: add a new job to set batch needs purge
	// e.g., batch.Job(models.JobTypeSetBatchNeedsStagingPurge),
	return models.QueueBatchJobs(batch,
		batch.Job(models.JobTypeValidateTagManifest, nil),
		models.NewJob(models.JobTypeSyncDir, args),
		batch.Job(models.JobTypeSetBatchStatus, makeBSArgs(models.BatchStatusPassedQC)),
	)
}

// QueueBatchGoLiveProcess fires off all jobs needed to call a batch live and
// ready for archiving. These jobs should only be queued up after a batch has
// been ingested into the production ONI instance.
func QueueBatchGoLiveProcess(batch *models.Batch, batchArchivePath string) error {
	var finalPath = filepath.Join(batchArchivePath, batch.FullName())
	return models.QueueBatchJobs(batch,
		models.NewJob(models.JobTypeSyncDir, makeSrcDstArgs(batch.Location, finalPath)),
		models.NewJob(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		batch.Job(models.JobTypeSetBatchLocation, makeLocArgs("")),
		batch.Job(models.JobTypeMarkBatchLive, nil),
	)
}
