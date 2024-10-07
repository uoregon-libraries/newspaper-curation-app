---
title: Technical Details
weight: 10
description: Deeper explanation of NCA's various workflows
---

This document attempts to explain the entire workflow from upload to batch
generation in a way that developers can understand what's needed and how to at
least begin investigating if something goes wrong.

## Jobs and the Job Queue

All background work in NCA is made up of relatively small parts tied together
in a single "pipeline". A pipeline represents a distinct operation that is made
up of smaller units, the jobs themselves. A job is usually the smallest atomic
"thing" we can run: updating an issue status in the database, calling out to
openjpeg to generate JP2 derivatives from an issue's PDFs, etc. We attempt to
make all jobs idempotent: running a job that already ran should never change
the database / file system / app state.

The pipeline organizes jobs into the more complex operations. For instance,
when it's time to pull PDFs from SFTP into NCA, that generates a pipeline
consisting of over a dozen atomic jobs: things like updating the issue's status
so NCA knows it's being worked on, copying the files to the workflow location,
splitting pages, etc. Even the "move files" operations are idempotent: the copy
is one job, then a job verifies that the copied files are correct, and then a
third job removes the source files.

The job runner, started by the `run-jobs` binary, regularly scans the database
looking for jobs to run. The default setup has different queues to keep
I/O-heavy jobs, such as derivative generation, from delaying fast jobs like
small database updates. This makes NCA more efficient, as jobs can run in
parallel when there won't be resource contention. Jobs in the same pipeline
will never be run in parallel, as it's assumed there are dependencies from one
to the next, but when multiple pipelines are queued up, NCA will process
whatever is next in each pipeline.

If you're trying to watch job logs as a whole, this can be confusing: a
pipeline's jobs will run in their sequence, but different pipelines can be
running at the same time, so job logs can look chaotic. If you're trying to
watch jobs for a given operation, you'll want to group them by pipeline to make
sense of what's going on.

The job runner also looks for issues in the scan and page review areas that are
ready to enter the workflow. These aren't actual jobs and aren't tied to
pipelines, they're just a separate background task that's always being watched.

All jobs store logs in the database, but these are currently not exposed to end
users (not even admins). To help mitigate this, the job runner also logs to
STDERR, though without pipeline filtering, those again can be tricky to parse
without some advanced log filtering application.

## Uploads

Whenever issues are uploaded into NCA, the application's "Uploaded Issues"
pages will display these issues along with any obvious errors the application
was able to detect. After a reasonable amount of time (to ensure uploading is
completed; some publishers slowly upload issue pages throughout the day, or
even multiple days), issues may be queued up for processing. Too-new issues
will be displayed, but queueing will be disabled.

Born-digital issues, when queued, are preprocessed (in order to ensure
derivatives can be generated, forcing one-pdf-per-page, etc.), then moved into
the page review area. The pages will be named sequentially in the format
`seq-dddd.pdf`, starting with `seq-0001.pdf`, then `seq-0002.pdf`, etc. These
PDFs might already be ordered correctly, but we've found the need to manually
reorder them many times, and have decided an out-of-band process for reviewing
and reordering is necessary. An easy approach is to have somebody use
something like Adobe Bridge to review and rename in bulk. Once complete, an
issue's pages need to be ordered by their filenames, e.g., `0001.pdf`,
`0002.pdf`, etc. Until issues are all given a fully numeric name, the job
runner will not pick them up.

**Note**: if issue folders are deleted from the page review location for any
reason, they must be cleaned up manually: [Handling Page Review Problems][1].
Once NCA is tracking uploads, deleting them outside the system will cause error
logs to go a bit haywire, and the issues can't be re-uploaded since NCA will
believe they quasi-exist.

For scanned issues, since they are in-house for us, it is assumed they're
already going to be properly named (`<number>.tif` and `<number>.pdf`) and
ordered, so after being queued they immediately get moved and processed for
derivatives.

The bulk upload queue tool (compiled to `bin/bulk-issue-queue`) can be used to
push all issues of a given type (scan vs. born digital) and issue key into the
workflow as if they'd been queued from the web app. This tool should only be
run when people aren't using the NCA queueing front-end, as it will queue
things faster than the NCA cache will be updated, which can lead to NCA's web
view being out of sync with reality. The data will be intact, but it can be
confusing. Also note that for scanned issues, this tool can take a long time
because it verifies the DPI of all images embedded in PDFs.

[1]: <{{% ref "handling-page-review-problems" %}}>

## Derivative Processing

Once issues are ready for derivatives (born-digital issues have been queued,
pre-processed, and renamed; scanned issues have been queued and moved), a job
is queued for derivative processing. This creates JP2 images from either the
PDFs (born-digital) or TIFFs (scanned), and the ALTO-compatible OCR XML based
on the text in the PDF. In our process, the PDFs are created by OCRing the
TIFFs. This process is manual and out-of-band since we rely on Abbyy, and
there isn't a particularly easy way to integrate it into our workflow.

The derivative generation process is probably the slowest job in the system.
As such, it is particularly susceptible to things like server power outage. In
the event that a job is canceled mid-operation, somebody will have to modify
the database to change the job's status from `in_process` to `pending`.

The derivative jobs are very fault-tolerant:

- Derivatives are generated in a temporary location, and only moved into the
  issue folder after the derivative has been generated successfully
- Derivatives which were already created are not rebuilt

These two factors make it easy to re-kick-off a derivative process without
worrying about data corruption.

Note that different OSes can report to NCA that something worked when the OS
still has yet to fully sync the files. This is out of NCA's control, and it is
exceedingly rare that it causes problems, but really unusual events (like a
very unfortunately-timed power failure, or catastrophic OS crash) can leave
things in a state that causes problems which NCA can't do anything about. These
kinds of events are virtually nonexistent even when power failures occur, but there are ways to help prevent problems:

- Make sure your system has a UPS so small power failures don't cause problems.
- Make sure your system's got enough disk space! Disk exhaustion is one of the
  worst problems that even modern OSes still handle very poorly.
- Replace faulty hardware! A hard-crash that's bad enough can interrupt a
  process before the OS has a chance to finalize file I/O.

## Error Reports

If an issue has some kind of problem which cannot be fixed with metadata entry,
the metadata person will report an error. Once an error is reported, the issue
will be hidden from all but Issue Managers in the NCA UI and one of them will
have to decide how to handle it. See [Fixing Flagged Workflow Issues][2].

[2]: <{{% ref "fixing-flagged-workflow-issues" %}}>

## Post-Metadata / Batch Generation

After metadata has been entered and approved, the issue is considered "done".
An issue XML will be generated (using the METS template defined by the setting
`METS_XML_TEMPLATE_PATH`) and born-digital issues' original PDFs are moved into
the issue location for safe-keeping. Assuming these are done without error, the
issue is marked "ready for batching".

A "batch builder" can then select organizations (e.g., the MARC org codes) they
want batches built for by visiting the "Create Batches" page in NCA. General
high-level aggregate data should give the batch builder enough information to
choose what to batch, after which they decide how big the batches should be.

Alternatively, the batch queue command-line script (compiled to
`bin/queue-batches`) grabs all issues which are ready to be batched, organizes
them by organization (a.k.a., MARC Org Code / awardee) for batching (*each
awardee must have its issues in a separate batch*), and generates batches if
there are enough pages (see the `MINIMUM_ISSUE_PAGES` setting).

**Note**: the `MINIMUM_ISSUE_PAGES` setting will be ignored if any issues
waiting to be batched have been ready for batching for more than 30 days. This
is necessary to handle cases where an issue had to have special treatment after
the bulk of a batch was completed, and would otherwise just sit and wait
indefinitely.

## Batch Management

Once a batch is queued for generation:

- The files will be put into the configured `BATCH_OUTPUT_PATH`
- The live files (non-TIFF, non-tar originals, etc.) are synced to the
  `BATCH_PRODUCTION_PATH`
- NCA sends a command to the staging ONI Agent (configured via `STAGING_AGENT`)
  to load the batch
- NCA polls the agent until the batch load is reported as successful

Once all these jobs are complete, the batch will be visible to batch reviewers,
letting them know action is needed to approve the batch. The batch page will
have a link to the staging environment's batch page for easier review, as well
as two possible actions to take: approve the batch for production or reject it
from staging due to problems in one or more issues.

If rejected, batch reviewers will need to find and flag the problem issues so
NCA can process the rest of the batch. Issues will be flagged as unfixable
(moving to a state where issue managers will have to take action), and the
batch reviewer will need to enter a comment to help identify what was wrong.
Once issues are done being flagged, the batch reviewer can finalize the batch,
rebuilding it with only the good issues, and NCA will reload it on staging
where it will be ready for another round of QC.

Once a batch has been approved in staging, NCA will contact the production ONI
Agent (configured via `PRODUCTION_AGENT`) to load it live, and then poll the
agent regularly until the batch load has completed.

After batches are live, NCA will move the original batch and any backups (the
source PDFs for born-digital batches, for instance) to the archival location
specificed by the configured `BATCH_ARCHIVE_PATH`. At this point the batch is
ready for final archival.

## Batch Archival

At UO, our archive path is actually a "staging area" for batches which are
getting ready for a move to the dark archive. Your process may not be quite the
same, but hopefully this is of help even if only to understand why NCA handles
batches the way it does.

Once we have enough content to justify a push to the dark archive, we move a
pile of batches from the staging area into the transfer area. Our archival team
runs the process (they generate their own manifests, for instance, and sync all
files to the archive). When they confirm that batches are archived, we flag
them in NCA as such.

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
