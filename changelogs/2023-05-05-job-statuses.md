### Fixed

- All issue- and batch-specific jobs are first setting the object's state and
  saving it, and only on success queueing up jobs. This fixes rare issues where
  a slow or dead job runner would allow a user to try to take action on an
  issue/batch that was already scheduled to have a different action taken.
  Rare, but disastrous.

### Added

- More in-depth documentation and recipes for manual testing. (See the `test/`
  directory's `README.md`)
- A new job type for canceling other jobs. This is strictly for purging jobs
  that failed permanently, or were on hold waiting for a failed job. This kind
  of job will be created by `purge-dead-issues` now.

### Changed

- The `jobs` package no longer exposes a bunch of low-level functionality so
  that the app as a whole is more predictable. Nothing outside `jobs` can just
  toss random jobs into a queue without creating a high-level function.
- The `purge-dead-issues` command:
  - Delegates all job creation and processing to `jobs` rather than having some
    job-processor calls and some immediate calls to make changes.
  - *No longer does a dry run*, and this is no longer even an option.
  - No longer generates a JSON report of what occurred.
