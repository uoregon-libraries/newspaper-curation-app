# NCA Changelog

All notable changes to NCA will be documented in this file.

Starting from NCA v2.10.0, a changelog shall be kept based loosely on the
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.  Since this is
an internal project for UO, we won't be attempting to do much in the way of
adding helpful migration notes, deprecating features, etc.  This is basically
to help our team keep up with what's going on.  Ye be warned.  Again.

<!-- Template

## vX.Y.Z

Brief description, if necessary

### Fixed

### Added

### Changed

### Removed

### Migration
-->

## v2.12.1

### Fixed

- Deploy script works with the version of goose we recommend using.  Oops.

### Changed

- Exponential backoff of failed jobs is now smarter to give failures more time
  to resolve before the first retry
- Invalid filenames are now checked more carefully before issues can get into
  NCA, to avoid dev-only manual filesystem fixing

## v2.12.0

User persistence!

Better handling of filenames!

Issue comments!!!!!!11!!1!1!one!!!1

### Fixed

- Users are now deactivated rather than deleted, which fixes a potential crash
  when viewing rejection notes or passing an issue back to the original
  curator's desk.  Honestly I'm not sure how this never crashed before.

### Added

- Issues can now have comments, and those comments are visible so long as the
  issue is part of NCA's workflow.  They're normally optional, but are required
  when rejecting an issue from the metadata queue (comments replace the
  previous one-off rejection note)
- New top-level `manage` script to simplify development and testing

### Changed

- Filenames for scanned and renamed PDF pages no longer have to be exactly four
  digits (e.g., `0001.pdf`).  As long as they're purely numeric, they're
  considered usable.  This fixes a long-standing problem where pages could be
  accidentally renamed to be three digits, and the process to get them fixed is
  annoying and confusing to the end user.
- Updated `uoregon-libraries/gopkg` to v0.15.0 to get a numeric file sort into NCA

### Removed

- Some dead / obsolete things have been dropped:
  - Removed a dead database table we haven't used in years
  - Removed unused `db/dbconf-example.yml` (this isn't necessary for the new
    fork of `goose` we've been recommending ... for a long time)
  - Removed dead "helper" SQL for identifying issues that can be manually
    deleted from the filesystem (now that there's a tool to handle this, the
    helper sql was just confusing)

### Migration

- Migrate the database, e.g, with `goose`: `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
- Delete the NCA cache, e.g., `rm /tmp/nca/* -rf` - this may not be necessary,
  but with how much the data has changed and how fast NCA is to regenerate it
  (under ten minutes when reading our 1.3 million live pages), it's worth a
  little time for a bit of extra safety.

## v2.11.4

Multi-awardee titles and some fixes

### Fixed

- Updated gopkg to v0.14.1, which should further reduce permissions issues for
  copied files/dirs
- Scanned titles in more than one MARC Org Code directory will now be visible
  and can be queued via the NCA UI
- Buggy page number labeling in templates is fixed
- Batch queueing now shows a proper error when it fails
- Widened a DB column which was causing jobs to fail to be created (very rarely)

### Added

- `delete-live-done-issues` is now deployed to production

### Migration

- Migrate the database, e.g, with `goose`: `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`

## v2.11.3

"Thumbs.db still sucks.  So do permissions."

### Fixed

- Page-review mover also ignores Thumbs.db
- Page-split job gives read/execute permissions to the world on its directory
  to hopefully finally really really fix permissions problems

## v2.11.2

"Thumbs.db sucks"

### Fixed
- File validation ignores Thumbs.db
- File cleaner removes Thumbs.db

## v2.11.1

Hotfix for UI issues

### Fixed

- No more JS errors when tabs aren't present
- All tables that have sortable attributes will get sort buttons

## v2.11.0

2.11 includes a major rewrite to the jobs engine, with a few other updates
sprinkled in.

### Fixed

- The uploads list in the front-end part of the application is now an HTML
  table, making it a lot easier to read, and fixing the odd flow problems for
  titles with long names
- Varous job-related problems have been addressed by the rewrite; see the
  "Changed" section below.

### Added

- The uploads list now shows a count of issues as well as an explanation of
  what the quick error scan actually means
- There's a new command to remove issues from disk that are associated with
  old `live_done` batches (batches which have been archived 4+ weeks ago) to
  avoid the risks of trying to identify and manually remove unneeded issues.
- There's a new, terrible, janky script: `test/report.sh`.  This script allows
  rudimentary testing of the database and filesystem states in order to act
  something like end-to-end testing for major refactors.

### Changed

- Background jobs are split up into more, but smaller, pieces.  When (not if)
  something goes wrong, it should be a lot easier to debug and fix it.
- Due to the jobs now being idempotent, all will attempt to auto-retry on
  failure.  This should mean no more having to monitor for temporary failures
  like an openjpeg upgrade going wrong, the database being restarted, NFS
  mounts failing, etc.
- `make bin/*` runs are a bit faster now, and `make` is much faster

### Migration

- All jobs have to be finished before you can upgrade from previous versions,
  because many major changes happened to the jobs subsystem.  This will require
  a few manual steps:
  - Turn off the server and worker processes.  This ensures that nobody is
    changing data, and no uploads will be processed.
  - Check the list of job types in the database:
    `SELECT DISTINCT job_type FROM jobs WHERE status NOT IN ('success', 'failed_done');`
  - Run workers for each outstanding job type, e.g., `./bin/run-jobs -c ./settings watch build_mets`
  - Repeat until no more outstanding jobs are in the database
- Run migrations prior to starting the services again

## v2.10.0

The big change here is that we don't force all titles to claim they're in
English when we generate ALTO.  Yay!

### Fixed

- JP2 output should be readable by other applications (file mode was 0600
  previously, making things like RAIS unable to even read the JP2s without a
  manual `chmod`)
- The check for a PDF's embedded image DPI is now much more reliable, and has
  unit testing

### Added

- Multi-language support added:
  - We read the 3-letter language code from the MARC XML and use that in the
    Alto output, rather than hard-coding ALTO to claim all blocks are English
  - Please keep in mind: this doesn't deal with the situation where a title can
    be in multiple languages - we use the *last* language we find in the MARC
    XML, because our internal process still has no way to try and differentiate
    languages on a per-text-block basis.
