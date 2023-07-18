### Added

- Database migrations are now easier to run and more self-contained in a new
  binary that reads DB settings from your configuration.

### Changed

- Background jobs have been fundamentally changed:
  - All jobs belong to a "pipeline"
  - A pipeline is a group of jobs built to accomplish a given high-level
    process in NCA, such as preparing a PDF for page renumbering, building a
    batch out of a set of issues, etc.
  - Non-devs shouldn't notice any change in NCA!
  - Devs will be able to query the database to more easily find grouped jobs
    for debugging, seeing what's still pending for a pipeline they're waiting
    on, etc.
  - Eventually we hope this helps us create a UI where you can see a pipeline
    and all its related jobs and their statuses, runtime, etc.

### Migration

- Drain the job queue entirely! This means no jobs should be in any status
  other than `success` or `failed_done`.
  - Check the database manually: `SELECT * FROM jobs WHERE status NOT IN
    ('success', 'failed_done');`
  - Don't add anything new to the PDF / scanned issue folders
  - Wait for pending and on-hold jobs to complete
  - Requeue any `failed` jobs or else cancel them (e.g., with
    `purge-dead-issues`)
  - **Note**: If you leave any jobs in any status other than `success` or
    `failed_done`, the database migrations will refuse to run and you won't be
    able to start up the NCA server.
- Shut down NCA entirely, deploy the new version, and run the database
  migrations. *Note: this won't run if you don't drain the job queue first.*
  - `make && ./bin/migrate-database -c ./settings up`
