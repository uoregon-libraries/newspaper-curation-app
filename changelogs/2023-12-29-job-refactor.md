### Fixed

- Incorrect toggling of a batch state exposed a batch for processing (ready for
  quality control) when it still had jobs in the queue. It was extremely
  unlikely somebody would get to the batch and do anything with it in between
  jobs, but it was still a possibility.
- When a user rejects a batch, the process is now a bit more streamlined in the
  codebase, making database errors less annoying if they do occur.

### Added

- Dev: The job runner now has a special flag that runs a single job and then
  exits. This can help identify which job is going rogue: run a job, check
  database state, run the next job, check state, etc.

### Changed

- Dev: jobs now have a way to signal more than just success or temporary
  failure. This is primarily needed for the API jobs where we need a "not
  failed, but wait and retry" status while waiting on ONI to complete a task.
  But it will also (someday) allow us to make some jobs fail immediately when
  there's an obvious problem that can't be retried.
