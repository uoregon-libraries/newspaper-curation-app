### Added

- Various instructions and status-setting buttons have been added to the batch
  management page for batch loaders
- Instructions for batch loaders' manual tasks now have a "copy" button, which
  should make manual tasks a bit easier
- Batches which were once on staging now have to be marked as purged from
  staging before they can move to other statuses (e.g., loading to production)

### Changed

- In the batch management handlers, inability to load a batch due to a database
  failure now logs as "critical", not just "error".

### Removed

- "Failed QC" has been removed as a batch status, as it is no longer in use

### Migration

- Do not update if you have batches in the `failed_qc` status. Get batches out
  of this status (e.g., by running the now-defunct batch fixer command-line
  tool), because it is no longer valid.
- Database migration, e.g.:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
