### Fixed

- Devs: `test/report.sh` strips more DB ids correctly, making test reporting
  far easier to scan for real changes

### Changed

- All jobs which were previously `SyncDir` have been split into two:
  - SyncRecursive is a light-weight, self-replicating job designed to take on
    the majority of what `SyncDir` used to do. For a given source and
    destination, all regular files (which don't exist or are a different file
    size) are copied without any post-copy validation. All directories are
    aggregated and queued up as new `SyncRecursive` jobs to be run as
    "siblings" (same priority) in the pipeline.
  - VerifyRecursive validates the SHA256 hash of every file in the source
    directory matches that in the destination directory, re-copying any which
    didn't. This is exatly the same as the prior `SyncDir` job, it just has
    less work to do since the new `SyncRecursive` job(s) will do the initial
    copying.
- Devs: made it so `test/recipes/general-test.sh` is able to "resume" its state
  to a certain degree, allowing for mistakes to be made when building a complex
  test up. This pattern should make things easier when time-consuming tasks
  need to be tested. See the script for details; this is really only important
  for devs working heavily on the NCA codebase.
