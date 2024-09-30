---
title: Workflow
weight: 30
description: Explanation of NCA's various workflows
---

This document details NCA's high-level workflow, hopefully to act as a guide to
generally understanding what goes on without necessarily worrying about all the
inner workings.

## Setup

See [Server Setup](/setup/server-setup) for getting the software installed, and
[Services](/setup/services) for information about running the services NCA
requires.

1. Server is set up, directories mounted
1. Settings file (`/usr/local/nca/settings`, for example) is customized as needed
1. Admin starts NCA in debug mode to create users
1. Title manager creates title records for all titles the app will see
1. MARC Org Code manager creates awardee codes NCA will need to know about

## Uploads

### SFTP (Born Digital)

1. Publishers upload PDF issues routinely to your servers
   - Uploads either go directly into NCA's SFTP folder, or a script can be built to move them
   - [See our detailed folder and filename specs](/specs/upload-specs)
1. Uploaded issues are individually verified and queued by a workflow manager using the "Uploaded Issues" section of the NCA web app
   - (Or the bulk queue CLI script is run if issues are verified out-of-band or trusted implicitly)
1. The job runner picks up queued issues:
   - Issues are pre-processed to ensure they can be read properly
   - Issues are split so there is exactly one PDF per page of the issue
   - Issues are then moved to the "page review" area for manual processing
1. Somebody reviews issues in the page review area:
   - Files must be renamed to indicate the review is complete (e.g., "seq-0001.pdf" to "0001.pdf")
   - Files may be reordered if necessary
   - If there are invalid PDFs, they may be deleted
   - If the "issue" actually contains two issues, the secondary issue's files should be removed and reuploaded in the correct folder
   - **If the entire issue is broken and needs to be removed from the system, developer involvement is necessary**
1. After files are reordered:
   - They must not be touched for a while, to ensure renaming/manipulation is complete
   - The job runner moves the files out of the page review folder and into the internal folder structure
   - Derivatives are created so the issue has the expected ALTO XML and JP2 files

### Scanned in-house

1. Digital imaging personnel scan papers and run them through OCR to produce a TIFF and PDF file
   - We use Abbyy for scanning, and the output PDF works with NCA
   - [See our detailed folder and filename specs](/specs/upload-specs)
1. Issues' PDFs and TIFFs are uploaded
   - Uploads either go directly into NCA's scans folder, or a script can be built to move them
1. The job runner automatically searches the scans folder for issues ready to move into the workflow:
   - They must not be touched for a while, to ensure all manipulation is complete
   - The job runner moves the files out of the scans folder and into the internal folder structure
   - Derivatives are created so the issue has the expected ALTO XML and JP2 files
     - In the case of scans, the JP2 is built from the TIFF, not the PDF

## Preparing Issues for Batching

After issues have been moved to the internal folders, and have had derivatives
generated, the workflow is the same regardless of the source:

1. An issue curator enters metadata for the issue and queues it for review
2. An issue reviewer validates the metadata and rejects it or approves it
3. Once metadata is entered and approved, the issue has its final derivative
   generated (METS XML) and awaits batching
4. When enough issues are ready, there are two ways to generate batches:
   - A dev can use the `queue-batches` command, generating batches for all
     issues which are ready.
   - Somebody with the "batch builder" role can visit NCA's "Create Batches"
     page and choose which MOCs should have issues batched.
5. Batches will be put into the configured `BATCH_OUTPUT_PATH`, and required
   files (e.g., not TIFFs) will be synced to production (as configured via
   `BATCH_PRODUCTION_PATH`).
6. A batch loader must manually load the batches into a staging server and then
   flag them as ready for QC
7. A batch reviewer does quality control on the staging server, verifying the
   batch's issues look good
   - If all is well, batch reviewer marks the issue as ready for production
   - If not, they reject the batch and can then get individual issues pulled
     out to be re-curated or rejected from NCA entirely. The remaining issues
     are put into a new batch which is then set as being ready for staging
     (repeating step 5).
8. Batches that are ready for production get loaded to prod by a batch loader.
9. Batch loader flags the batch as "live", and NCA moves its files to the
   configured `BATCH_ARCHIVE_PATH`.
10. Once the batch can be confirmed as fully archived, a batch loader flags it
    as archived.
11. Somebody with command-line access to NCA will run `delete-live-done-issues`
    (or set up a cron job) to clean unneeded files from NCA that are part of
    archived batches. It only deletes files when a batch is at least four weeks
    past its archive date to ensure any final problems can be handled.
