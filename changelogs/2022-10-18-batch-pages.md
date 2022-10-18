### Fixed

- Batches that have issues removed will no longer "stall" in the job queue
  (`Batch.AbortIssueFlagging` allows pending batches now in addition to those
  flagged as needing QC)

### Added

- The `manage` script restarts key services after shutting them down

### Changed

- More intuitive redirects from batch management pages
