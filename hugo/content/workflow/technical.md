---
title: Technical Details
weight: 10
description: Deeper explanation of NCA's various workflows
---

This document attempts to explain the entire workflow from upload to batch
generation in a way that developers can understand what's needed and how to at
least begin investigating if something goes wrong.

## Jobs and the Job Queue

The job runner regularly scans the database looking for jobs to run.  The
default setup splits jobs up to ensure quick jobs, like moving an issue from
one location to another on the filesystem, are run separately from slow jobs
like generating JP2 files.  This ensures that slow jobs don't hold up the
faster jobs, but could be confusing if you're expecting to see jobs run in the
order they are queued.  It also tends to make raw job logs confusing.

The job runner also looks for issues in the page review area that have been
renamed and are ready to enter the workflow.

All jobs store logs in the database, but these are currently not exposed to end
users (not even admins).  To help mitigate this, the job runner also logs to
STDERR, so those can be captured and reviewed.

## Uploads

Whenever issues are uploaded into NCA, the application's "Uploaded Issues"
pages will display these issues along with any obvious errors the application
was able to detect.  After a reasonable amount of time (to ensure uploading is
completed; some publishers slowly upload issue pages throughout the day, or
even multiple days), issues may be queued up for processing.  Too-new issues
will be displayed, but queueing will be disabled.

Born-digital issues, when queued, are preprocessed (in order to ensure
derivatives can be generated, forcing one-pdf-per-page, etc.), then moved into
the page review area.  The pages will be named sequentially in the format
`seq-dddd.pdf`, starting with `seq-0001.pdf`, then `seq-0002.pdf`, etc.  These
PDFs might already be ordered correctly, but we've found the need to manually
reorder them many times, and have decided an out-of-band process for reviewing
and reordering is necessary.  An easy approach is to have somebody use
something like Adobe Bridge to review and rename in bulk.  Once complete, an
issue's filenames need to be ordered by their filenames, e.g., `0001.pdf`,
`0002.pdf`, etc.  Until issues are all given a fully numeric name, the job
runner will not pick them up.

**Note**: if issue folders are deleted from the page review location for any
reason, they must be cleaned up manually:
[Handling Page Review Problems](/workflow/handling-page-review-problems).  Once
NCA is tracking uploads, deleting them outside the system will cause error logs
to go a bit haywire, and the issues can't be re-uploaded since NCA will believe
they quasi-exist.

For scanned issues, since they are in-house for us, it is assumed they're
already going to be properly named (`<number>.tif` and `<number>.pdf`) and
ordered, so after being queued, they get moved and processed for derivatives,
then they're available in the workflow for metadata entry.

The bulk upload queue tool (compiled to `bin/bulk-issue-queue`) can be used to
push all issues of a given type (scan vs. born digital) and issue key into the
workflow as if they'd been queued from the web app.  This tool should only be
run when people aren't using the NCA queueing front-end, as it will queue
things faster than the NCA cache will be updated, which can lead to NCA's web
view being out of sync with reality.  The data will be intact, but it can be
confusing.  Also note that for scanned issues, this tool can take a long time
because it verifies the DPI of all images embedded in PDFs.

## Derivative Processing

Once issues are ready for derivatives (born-digital issues have been queued,
pre-processed, and renamed; scanned issues have been queued and moved), a job
is queued for derivative processing.  This creates JP2 images from either the
PDFs (born-digital) or TIFFs (scanned), and the ALTO-compatible OCR XML based
on the text in the PDF.  In our process, the PDFs are created by OCRing the
TIFFs.  This process is manual and out-of-band since we rely on Abbyy, and
there isn't a particularly easy way to integrate it into our workflow.

The derivative generation process is probably the slowest job in the system.
As such, it is particularly susceptible to things like server power outage.  In
the event that a job is canceled mid-operation, somebody will have to modify
the database to change the job's status from `in_process` to `pending`.

The derivative jobs are very fault-tolerant:

- Derivatives are generated in a temporary location, and only moved into the
  issue folder after the derivative has been generated successfully
- Derivatives which were already created are not rebuilt

These two factors make it easy to re-kick-off a derivative process without
worrying about data corruption.

## Error Reports

If an issue has some kind of problem which cannot be fixed with metadata entry,
the metadata person will report an error.  Once an error is reported, the issue
will be hidden from all but Issue Managers in the NCA UI and one of them will
have to decide how to handle it.  See
[Fixing Flagged Workflow Issues](/workflow/fixing-flagged-workflow-issues).

## Post-Metadata / Batch Generation

After metadata has been entered and approved, the issue is considered "done".
An issue XML will be generated (using the METS template defined by the setting
`METS_XML_TEMPLATE_PATH`) and born-digital issues' original PDF(s) is/are moved
into the issue location for safe-keeping.  Assuming these are done without
error, the issue is marked "ready for batching".

The batch queue command-line script (compiled to `bin/queue-batches`) grabs all
issues which are ready to be batched, organizes them by MARC Org Code (a.k.a.,
awardee) for batching (*each awardee must have its issues in a separate
batch*), and generates batches if there are enough pages (see the
`MINIMUM_ISSUE_PAGES` setting).

**Note**: the `MINIMUM_ISSUE_PAGES` setting will be ignored if any issues
waiting to be batched have been ready for batching for more than 30 days.  This
is necessary to handle cases where an issue had to have special treatment after
the bulk of a batch was completed, and would otherwise just sit and wait
indefinitely.

## Batch Management

Once a batch is generated and all jobs related to it are complete, the files
will be put into the configured `BATCH_OUTPUT_PATH` and the "Batches" page in NCA
will show it to users with the "batch loader" role.

At this point the batch can be loaded into staging. NCA's batch page,
accessible by activating the relevant link in the batch list, will use your
configuration to provide bash commands that batch loaders can copy and paste in
order to get the batch onto your staging ONI instance.

*Note: if your staging system mounts files differently than your NCA
server, the commands may have to be altered. e.g., NCA might use
`/mnt/news/outgoing` while staging uses `/mnt/libnca`.*

Once loaded onto staging, the batch loader flags a batch as being ready for QC
(quality control). After some processing, the batch will be visible to batch
reviewers in NCA. The batch page will have a link to the staging environment's
batch page for easier review, as well as two possible actions to take: approve
the batch for production or reject it from staging due to problems in one or
more issues.

If rejected, batch reviewers will need to find and flag the problem issues so
NCA can process the rest of the batch. Issues will be flagged as unfixable
(moving to a state where issue managers will have to take action), and the
batch reviewer will need to enter a comment to help identify what was wrong.
Once issues are done being flagged, the batch reviewer can finalize the batch,
rebuilding it with only the good issues, and moving it into the "ready for
staging" state, where a batch loader is guided through purging and reloading
the batch for another round of QC.

Once a batch has been approved in staging, all essential files (e.g., no TIFFs)
will be copied to the configured `BATCH_PRODUCTION_PATH` location and then NCA
will mark the batch as ready to go live. Upon visiting the batch page in NCA,
the batch loader will get instructions for purging the batch from staging and
then loading the batch to production. *The same caveat applies here as when
loading to staging: if file mounts differ from NCA's mount locations, the batch
loader will need to adjust the commands NCA provides.*

After batches are live, the batch loader flags it as such in NCA, and NCA will
move the original batch and any backups (the source PDFs for born-digital
batches, for instance) to the archival location specificed by the configured
`BATCH_ARCHIVE_PATH`. At this point the batch is ready for final archival.

## Batch Archival

At UO, our archive path is actually a holding tank as we move things to the
final archive location in large batches rather than every time something is
considered "done". Your process may not be quite the same, but hopefully this
is of help even if only to understand why NCA handles batches the way it does.

Once we have enough content to justify a push to the dark archive, our archival
team runs the process (they generate their own manifests, for instance, and
sync all files to the archive). When batches have been confirmed as being in
our final archive location, we flag them in NCA as being archived.

Whether or not you follow that process, you will still need to specify an
archive path (`BATCH_ARCHIVE_PATH`) in your `settings` file and flag batches as
being live. Your archive path may be a direct mount to your final archive, a
location you manage manually, or some dummy location you simply delete if you
aren't preserving the original content for some reason.

Once flagged as archived, NCA stores the archival date and time. This is
important for knowing when it's safe to clean up the files. The issue deletion
script (`bin/delete-live-done-issues`, created by a standard `make` run) will
look for batches archived more than **four weeks ago** and then *completely
delete all files in NCA tied to these batches*. The files in your archive will
not be removed, of course, but NCA will ensure its workflow directories are
cleaned up to make room for new incoming files.

The four-week "timer" was originally put in place to ensure files have had a
chance to be fully backed up offsite, but it also serves another purpose: *it
gives you a chance to handle problems that weren't caught during the QC
process*. Once NCA's workflow files are removed, reprocessing a batch becomes
significantly more difficult.

Unless you have very few batches, very small batches, or a lot of disk space,
`bin/delete-live-done-issues` should be run regularly to avoid running out of
storage. NCA can handle most problems gracefully, but running out of storage
is almost guaranteed to cause you some headaches.
