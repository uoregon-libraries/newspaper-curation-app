---
title: Server Setup
weight: 10
description: Setting up a new NCA toolsuite
---

## Human Requirements

Unfortunately, this process is still technical enough that you will need a
devops person to at least get the system set up for processing.  You'll also
need people who can reorder PDF pages (if necessary) as well as people who can
enter and review newspaper issue metadata.

Somebody will want to monitor the output from the various automated processes,
such as QCing generated batches on a staging server prior to loading in
production, as there is still a great deal of room for human error.

## Preliminary setup

Before anything can be done, the following setup has to happen:

1. Make sure you understand the [Services](/setup/services) documentation and
   can get the stack up and running
1. Somebody symlinks or otherwise sets up the sftp folder root so that each
   title has its own location directly off said root.  e.g.,
   `/mnt/news/sftp/foo` should contain one title's issues, and
   `/mnt/news/sftp/bar` should contain a different title's issues.
1. Somebody sets up the full swath of folders, mounting to network storage
   as it makes sense.
   - `PDF_UPLOAD_PATH` (`/mnt/news/sftp`): One subfolder should be set up per title
   - `SCAN_UPLOAD_PATH` (`/mnt/news/scans`): This is where in-house scans would be uploaded.
   - `ORIGINAL_PDF_BACKUP_PATH` (`/mnt/news/backup/originals`): Short-term storage
     where uploaded PDFs will be moved after being split.  They may need to be
     held a few months for embargoed issues, but they're auto-purged once the
     issue has been put into a batch.
   - `PDF_PAGE_REVIEW_PATH` (`/mnt/news/page-review`): Issues which came from
     born-digital SFTP uploads and are ready for manual page reordering - this
     should be exposed to whomever will manually review and reorder the
     born-digital uploads prior to them entering the rest of the workflow.
   - `BATCH_OUTPUT_PATH` (`/mnt/news/outgoing`): Batches here are ready for
     ingest into staging and eventually production
   - `WORKFLOW_PATH` (`/mnt/news/workflow`): Issues are moved here for
     processing, and once here should never be accessible to anybody to
     manually modify them.  They will live here until all workflow tasks are
     complete and they're put into a batch for ingest.
   - `ISSUE_CACHE_PATH` (`/var/local/news/nca/cache`): This just needs to be
     created.  The app will use this to speed up issue lookups.
1. Make sure that the workflow path and the batch output path are on the same
   filesystem!  This ensures the batch generator will be able to hard-link
   files, rather than copying them, which saves a significant amount of time
   when building large batches.  **NOTE**: the system currently *requires*
   this, and will fail if an attempt to hard-link files fails.
1. Permissions have to be set up such that:
   - Humans can rename PDFs in the page review path
   - Humans can drop off scanned PDF/TIFF pairs in the scans path
   - Humans can upload born-digital PDFs into the sftp path
   - All binaries (`server`, `run-jobs`, anything else in `bin/` you wish to
     run) are run as a user who can read and write to all paths
   - Apache can read the scans path
   - The system which ingests batches into ONI can read from the batch
     output path
1. Run the servers and set up one or more users: [User Setup](/setup/user-setup)
1. Somebody must set up the list of newspaper titles using the "Manage
   Newspaper Titles" functionality.  Nothing works if titles aren't set up!
   Titles need all data except the username and password, which are primarily
   there to help keep the information central.
1. Somebody has to set up at least one MARC Org Code in the admin app's "MARC
   Org Codes" area.  This should match the code set up in the app's settings.
   If in-house scanning is done, and awardees will differ from your primary
   awardee's code, you would set up those awardees before putting their scanned
   images into the scan folder.
