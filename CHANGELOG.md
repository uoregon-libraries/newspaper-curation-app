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
-->

<!-- Unreleased: finalize and uncomment this in the release/* branch

## (Unreleased)

Brief description, if necessary

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

### Fixed

- Uploads list is now a table, making it a lot easier to read, and fixing the
  odd flow problems for titles with long names

### Added

- Uploads list now shows a count of issues as well as an explanation of what
  the quick error scan actually means

### Changed

- Background jobs are split up into more, but smaller, pieces.  When (not if)
  something goes wrong, it should be a lot easier to debug and fix it.  This
  also paves the way for a more resilient and automated retry when jobs fail
  due to something temporary like an openjpeg upgrade going wrong, the database
  being restarted, NFS mounts failing, etc.

### Removed

-->

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
