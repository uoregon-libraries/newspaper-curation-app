### Added

- A new setting, `BATCH_ARCHIVE_PATH`, has been introduced. Set this to the
  location NCA should move your batches after they're live.
- Batch loaders can now mark issues as live, which moves them to the
  aforementioned location, and as archived, which allows
  `bin/delete-live-done-issues` to remove their workflow files (after a delay).
- New documentation created to help devs creating new configuration settings.

### Changed

- Massive overhaul of workflow and batch management documentation to match the
  new processes

### Migration

- Get every pending batch out of NCA and into your production systems,
  otherwise batches might get processed incorrectly.
