### Added

- A new setting, `BATCH_PRODUCTION_PATH`, has been introduced. Set this to the
  location NCA should copy your batches when they're ready for being ingested
  into production.
- On QC approval, batches are automatically synced to the location specified by
  the new setting (`BATCH_PRODUCTION_PATH`).
