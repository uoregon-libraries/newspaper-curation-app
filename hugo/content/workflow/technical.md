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

Once batches are generated, they will appear in the configured
`BATCH_OUTPUT_PATH`.  The `batches` table in the database will show the batch
with a `status` of `qc_ready`.

Please note that a bagit job will still be running in the background.  Bag
files are unnecessary to load a batch into ONI or Chronam, so the job can
happen while somebody is reviewing the batch on a staging server, but the batch
should **not be considered production-ready** until the bagit files are
generated.  You can monitor the status of the job in the database directly, or
just watch for a valid tag manifest file.

If the batch has any bad issues, it must be [fixed](/workflow/fixing-batches)
with a command-line tool and then rebatched.

Once the batch has been approved in staging, (TODO: another utility!) run the
[manual go-live](/workflow/batch-manual-golive) process to get the batch and
its issues to be properly recognized by the rest of NCA as no longer being part
of the workflow.
