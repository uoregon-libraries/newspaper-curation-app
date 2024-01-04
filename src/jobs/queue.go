package jobs

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Job argument names are constants to let us define arg names in a way
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

func makeWSArgs(ws schema.WorkflowStep) map[string]string {
	return map[string]string{JobArgWorkflowStep: string(ws)}
}

func makeBSArgs(bs string) map[string]string {
	return map[string]string{JobArgBatchStatus: string(bs)}
}

func makeLocArgs(loc string) map[string]string {
	return map[string]string{JobArgLocation: loc}
}

func makeForcedArgs() map[string]string {
	return map[string]string{JobArgForced: JobArgForced}
}

func makeSrcDstArgs(src, dest string) map[string]string {
	return map[string]string{
		JobArgSource:      src,
		JobArgDestination: dest,
	}
}

func makeActionArgs(msg string) map[string]string {
	return map[string]string{JobArgMessage: msg}
}

// getJobsForCopyDir combines the fast-copy job with the slow verify+recopy job
// so that all sync operations, even when not doing a full directory move, are
// as bulletproof as they can be.
func getJobsForCopyDir(source, destination string, exclusions ...string) []*models.Job {
	var args = makeSrcDstArgs(source, destination)
	args[JobArgExclude] = strings.Join(exclusions, ",")
	return []*models.Job{
		models.NewJob(models.JobTypeSyncRecursive, args),
		models.NewJob(models.JobTypeVerifyRecursive, args),
	}
}

// getJobsForMoveDir returns the list of jobs common to moving a directory:
//
//   - Copy files recursively, fast, and granularly (one job created per subdir)
//     to a "work in progress" location
//   - Sync dir - redundant, but verifies all files copied successfully long
//     enough after the copy to hopefully avoid any NFS / CIFS file caching that
//     reports things wrong. "Bad" copies should be rectified here.
//   - Kill old directory and all its files
//   - Rename work-in-progress directory to final directory
func getJobsForMoveDir(source, destination string, exclusions ...string) []*models.Job {
	// Get the parent dir of the destination so we can craft a WIP dir
	var dir, name = filepath.Split(filepath.Clean(destination))
	var wipDir = filepath.Join(dir, ".wip-"+name)
	var jobs = getJobsForCopyDir(source, wipDir, exclusions...)
	jobs = append(jobs, models.NewJob(models.JobTypeKillDir, makeLocArgs(source)))
	jobs = append(jobs, models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, destination)))

	return jobs
}

// QueueSFTPIssueMove queues up an issue move into the workflow area followed
// by a page-split and then a move to the page review area
//
// This process looks a bit weird.  What's in the issue location dir after page
// splitting is the original upload, which we back up since we may need to
// reprocess the PDFs from these originals.  Once we've backed up, we move the
// page-split files back into the proper workflow folder...  which is then
// promptly moved out to the page review area.
//
// TODO: Lots of fun jobs are involved in the "SFTP Issue Move" pipeline...
// this function (and the pipeline) probably need a new name.
func QueueSFTPIssueMove(issue *models.Issue, c *config.Config) error {
	var workflowDir = filepath.Join(c.WorkflowPath, issue.HumanName)
	var workflowPageSplitDir = filepath.Join(c.WorkflowPath, ".split-"+issue.HumanName)
	var pageReviewDir = filepath.Join(c.PDFPageReviewPath, issue.HumanName)
	var backupLoc = filepath.Join(c.PDFBackupPath, issue.HumanName)
	var jobs []*models.Job

	// Move dir and update issue location
	jobs = append(jobs, getJobsForMoveDir(issue.Location, workflowDir)...)
	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueLocation, makeLocArgs(workflowDir)))

	// Clean dotfiles and then kick off the page splitter
	jobs = append(jobs, models.NewJob(models.JobTypeCleanFiles, makeLocArgs(workflowDir)))
	jobs = append(jobs, issue.BuildJob(models.JobTypePageSplit, makeLocArgs(workflowPageSplitDir)))

	// Back up the original files and move the split files to the issue dir
	jobs = append(jobs, getJobsForMoveDir(workflowDir, backupLoc)...)
	jobs = append(jobs, models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(workflowPageSplitDir, workflowDir)))
	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueBackupLoc, makeLocArgs(backupLoc)))

	// Finally, sync the issue over to the page review location
	jobs = append(jobs, getJobsForMoveDir(workflowDir, pageReviewDir)...)
	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueLocation, makeLocArgs(pageReviewDir)))

	// It's ready for review! Easy!
	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueWS, makeWSArgs(schema.WSAwaitingPageReview)))
	jobs = append(jobs, issue.BuildJob(models.JobTypeIssueAction, makeActionArgs("Moved issue from SFTP into NCA")))

	return models.QueueIssueJobs(models.PNSFTPIssueMove, issue, jobs...)
}

// QueueMoveIssueForDerivatives creates jobs to move issues into the workflow,
// make all issues' pages numbered nicely, and then generate derivatives
func QueueMoveIssueForDerivatives(issue *models.Issue, workflowPath string) error {
	var workflowDir = filepath.Join(workflowPath, issue.HumanName)
	var jobs []*models.Job

	jobs = append(jobs, getJobsForMoveDir(issue.Location, workflowDir)...)
	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueLocation, makeLocArgs(workflowDir)))
	jobs = append(jobs, models.NewJob(models.JobTypeCleanFiles, makeLocArgs(workflowDir)))
	jobs = append(jobs, issue.BuildJob(models.JobTypeRenumberPages, nil))
	jobs = append(jobs, issue.BuildJob(models.JobTypeMakeDerivatives, nil))
	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueWS, makeWSArgs(schema.WSReadyForMetadataEntry)))
	jobs = append(jobs, issue.BuildJob(models.JobTypeIssueAction, makeActionArgs("Created issue derivatives")))

	return models.QueueIssueJobs(models.PNMoveIssueForDerivatives, issue, jobs...)
}

// QueueForceDerivatives will forcibly regenerate all derivatives for an issue.
// During the processing, the issue's workflow step is set to "awaiting
// processing", and only gets set back to its previous value on successful
// completion of the other jobs.
func QueueForceDerivatives(issue *models.Issue) error {
	var currentStep = issue.WorkflowStep
	return models.QueueIssueJobs(models.PNForceDerivatives, issue,
		issue.BuildJob(models.JobTypeMakeDerivatives, makeForcedArgs()),
		issue.BuildJob(models.JobTypeBuildMETS, makeForcedArgs()),
		issue.BuildJob(models.JobTypeSetIssueWS, makeWSArgs(currentStep)),
		issue.BuildJob(models.JobTypeIssueAction, makeActionArgs("Force-regenerated issue derivatives")),
	)
}

// QueueFinalizeIssue creates and queues jobs that get an issue ready for
// batching.  Currently this means generating the METS XML file and copying
// archived PDFs (if born-digital) into the issue directory.
func QueueFinalizeIssue(issue *models.Issue) error {
	// Some jobs aren't queued up unless there's a backup, so we actually
	// generate a list of jobs programatically instead of inline
	var jobs []*models.Job
	jobs = append(jobs, issue.BuildJob(models.JobTypeBuildMETS, nil))

	if issue.BackupLocation != "" {
		jobs = append(jobs, issue.BuildJob(models.JobTypeArchiveBackups, nil))
		jobs = append(jobs, models.NewJob(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)))
		jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueBackupLoc, makeLocArgs("")))
	}

	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueWS, makeWSArgs(schema.WSReadyForBatching)))
	jobs = append(jobs, issue.BuildJob(models.JobTypeIssueAction, makeActionArgs("Issue prepped for batching")))

	return models.QueueIssueJobs(models.PNFinalizeIssue, issue, jobs...)
}

// QueueMakeBatch sets up the jobs for generating a batch on disk: generating
// the directories and hard-links, making the batch XML, putting the batch
// where it can be loaded onto staging, and generating the bagit manifest.
// Nothing can happen automatically after all this until the batch is verified
// on staging.
func QueueMakeBatch(batch *models.Batch, c *config.Config) error {
	return models.QueueBatchJobs(models.PNMakeBatch, batch, getJobsForMakeBatch(batch, c)...)
}

// getJobsForMakeBatch returns all jobs needed to generate a batch. This is needed
// by two different higher-level tasks.
func getJobsForMakeBatch(batch *models.Batch, c *config.Config) []*models.Job {
	// Prepare the various directory vars we'll need
	var batchname = batch.FullName()
	var wipDir = filepath.Join(c.BatchOutputPath, ".wip-"+batchname)
	var outDir = filepath.Join(c.BatchOutputPath, batchname)
	var liveDir = filepath.Join(c.BatchProductionPath, batchname)

	var jobs []*models.Job

	// The first set of jobs builds the batch files in the batch output location
	jobs = append(jobs,
		batch.BuildJob(models.JobTypeCreateBatchStructure, makeLocArgs(wipDir)),
		batch.BuildJob(models.JobTypeSetBatchLocation, makeLocArgs(wipDir)),
		batch.BuildJob(models.JobTypeMakeBatchXML, nil),
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, outDir)),
		batch.BuildJob(models.JobTypeSetBatchLocation, makeLocArgs(outDir)),
		batch.BuildJob(models.JobTypeBatchAction, makeActionArgs("created batch")),
	)

	// Next comes the bag manifest files and a brief tagmanifest validation
	jobs = append(jobs,
		batch.BuildJob(models.JobTypeWriteBagitManifest, nil),
		batch.BuildJob(models.JobTypeBatchAction, makeActionArgs("wrote bagit manifest")),
		batch.BuildJob(models.JobTypeValidateTagManifest, nil),
	)

	// Finally, the last jobs copy the essential files to the final path so we
	// can ingest them into staging
	jobs = append(jobs, getJobsForCopyDir(outDir, liveDir, "*.tif", "*.tiff", "*.TIF", "*.TIFF", "*.tar.bz", "*.tar")...)
	jobs = append(jobs,
		batch.BuildJob(models.JobTypeBatchAction, makeActionArgs("copied to live path")),
		batch.BuildJob(models.JobTypeSetBatchStatus, makeBSArgs(models.BatchStatusStagingReady)),
	)

	return jobs
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
	return models.QueueIssueJobs(models.PNRemoveErroredIssue, issue, jobs...)
}

// QueuePurgeStuckIssue builds jobs for removing an issue that had critical
// failures on one or more jobs. Any waiting (on-hold) jobs still tied to the
// issue are removed, as are failed jobs, and then the issue is purged with
// data a dev can use to look into the problem more closely.
func QueuePurgeStuckIssue(issue *models.Issue, erroredIssueRoot string) error {
	var pipelines, err = issue.Pipelines()
	if err != nil {
		return fmt.Errorf("query pipelines for issue %d (%s): %s", issue.ID, issue.Key(), err)
	}

	var jobs []*models.Job
	var purgeReason = "Issue failed getting through workflow:\n"
	for _, p := range pipelines {
		purgeReason += fmt.Sprintf("- Pipeline %d (%s / %s):\n", p.ID, p.Name, p.Description)
		var list []*models.Job
		list, err = p.Jobs()
		if err != nil {
			return fmt.Errorf("query jobs on pipeline %d for issue %d (%s): %s", p.ID, issue.ID, issue.Key(), err)
		}
		for _, j := range list {
			switch models.JobStatus(j.Status) {
			case models.JobStatusFailed, models.JobStatusOnHold:
				if j.Status == string(models.JobStatusFailed) {
					purgeReason += fmt.Sprintf("  - Job %d (%s) failed too many times\n", j.ID, j.Type)
				}
				var jj = j.BuildJob(models.JobTypeCancelJob, nil)
				jobs = append(jobs, jj)
			}
		}
	}
	jobs = append(jobs, issue.BuildJob(models.JobTypeIssueAction, makeActionArgs(purgeReason)))
	jobs = append(jobs, getJobsForRemoveErroredIssue(issue, erroredIssueRoot)...)

	return models.QueueJobs(models.PNPurgeStuckIssue, fmt.Sprintf("Purging issue %s and its unfinished jobs", issue.Key()), jobs...)
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
	jobs = append(jobs, getJobsForMoveDir(issue.Location, contentDir)...)
	jobs = append(jobs, issue.BuildJob(models.JobTypeSetIssueLocation, makeLocArgs(contentDir)))
	jobs = append(jobs, issue.BuildJob(models.JobTypeMoveDerivatives, makeLocArgs(derivDir)))
	jobs = append(jobs, issue.BuildJob(models.JobTypeWriteActionLog, nil))

	// If we have a backup, archive it and remove its files
	if issue.BackupLocation != "" {
		jobs = append(jobs,
			issue.BuildJob(models.JobTypeArchiveBackups, nil),
			models.NewJob(models.JobTypeKillDir, makeLocArgs(issue.BackupLocation)),
			issue.BuildJob(models.JobTypeSetIssueBackupLoc, makeLocArgs("")),
		)
	}

	// Move to the final location and update metadata
	jobs = append(jobs,
		models.NewJob(models.JobTypeRenameDir, makeSrcDstArgs(wipDir, finalDir)),
		issue.BuildJob(models.JobTypeIgnoreIssue, nil),
		issue.BuildJob(models.JobTypeSetIssueLocation, makeLocArgs("")),
		issue.BuildJob(models.JobTypeSetIssueWS, makeWSArgs(schema.WSUnfixableMetadataError)),
		issue.BuildJob(models.JobTypeIssueAction, makeActionArgs("Errored issue removed from NCA")),
	)

	return jobs
}

// QueueBatchFinalizeIssueFlagging generates jobs for removing flagged issues
// from a batch which failed QC, then rebuilding the batch
func QueueBatchFinalizeIssueFlagging(batch *models.Batch, flagged []*models.FlaggedIssue, c *config.Config) error {
	// This is yet another set of jobs that has steps we build out rather than
	// just having a hard-coded list queued up
	var jobs []*models.Job

	// Destroy batch dir jobs - note that the batch dir contains hard links and
	// easily rebuilt metadata (e.g., the bagit info), so this is not truly a
	// destructive operation
	jobs = append(jobs,
		models.NewJob(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		batch.BuildJob(models.JobTypeSetBatchLocation, makeLocArgs("")),
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
			i.Issue.BuildJob(models.JobTypeRemoveFile, makeLocArgs(i.Issue.METSFile())),
			i.Issue.BuildJob(models.JobTypeFinalizeBatchFlaggedIssue, nil),
		)
	}

	// Remove all the no-longer-useful flagged issue data
	jobs = append(jobs, batch.BuildJob(models.JobTypeEmptyBatchFlaggedIssuesList, nil))
	jobs = append(jobs, batch.BuildJob(models.JobTypeBatchAction, makeActionArgs("removed flagged issues from batch")))

	// Regenerate batch
	jobs = append(jobs, getJobsForMakeBatch(batch, c)...)

	return models.QueueBatchJobs(models.PNFinalizeIssueFlagging, batch, jobs...)
}

// QueueBatchForDeletion is used when all issues in a batch need to be
// rejected, rendering the batch unnecessary (and useless).
func QueueBatchForDeletion(batch *models.Batch, flagged []*models.FlaggedIssue) error {
	// This is essentially a copy of the finalization job list, except there's no
	// regenerate-batch step
	var jobs []*models.Job

	// Destroy batch dir
	jobs = append(jobs,
		models.NewJob(models.JobTypeKillDir, makeLocArgs(batch.Location)),
		batch.BuildJob(models.JobTypeSetBatchLocation, makeLocArgs("")),
	)

	// Finalize flagged issues
	for _, i := range flagged {
		jobs = append(jobs,
			i.Issue.BuildJob(models.JobTypeRemoveFile, makeLocArgs(i.Issue.METSFile())),
			i.Issue.BuildJob(models.JobTypeFinalizeBatchFlaggedIssue, nil),
		)
	}

	// Remove all the no-longer-useful flagged issue data
	jobs = append(jobs, batch.BuildJob(models.JobTypeEmptyBatchFlaggedIssuesList, nil))

	// Destroy the batch
	jobs = append(jobs, batch.BuildJob(models.JobTypeDeleteBatch, nil))
	jobs = append(jobs, batch.BuildJob(models.JobTypeBatchAction, makeActionArgs("deleted batch")))

	return models.QueueBatchJobs(models.PNBatchDeletion, batch, jobs...)
}

// QueueBatchGoLiveProcess fires off all jobs needed to call a batch live and
// ready for archiving. These jobs should only be queued up after a batch has
// been ingested into the production ONI instance.
func QueueBatchGoLiveProcess(batch *models.Batch, batchArchivePath string) error {
	var finalPath = filepath.Join(batchArchivePath, batch.FullName())
	var jobs []*models.Job

	jobs = append(jobs, getJobsForMoveDir(batch.Location, finalPath)...)
	jobs = append(jobs, batch.BuildJob(models.JobTypeBatchAction, makeActionArgs("moved batch to archive location")))
	jobs = append(jobs, batch.BuildJob(models.JobTypeSetBatchLocation, makeLocArgs("")))
	jobs = append(jobs, batch.BuildJob(models.JobTypeMarkBatchLive, nil))

	return models.QueueBatchJobs(models.PNGoLiveProcess, batch, jobs...)
}
