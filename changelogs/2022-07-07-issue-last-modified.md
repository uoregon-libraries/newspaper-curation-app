### Fixed

- Issues will no longer be able to move into NCA while still being uploaded
  (see "Changed" section for details)

### Added

### Changed

- Major change to how an issue's "last modified" date is determined. Instead of
  relying on the files' creation/modification times, we now generate a manifest
  file that tells us what the files' sizes and checksums are at a given point
  in time. This will make NCA a lot slower when scanning issues, but some
  filesystem copy operations don't seem to properly tell us when the file was
  first copied, instead reporting the file's original creation time. This
  results in NCA thinking an issue that's mid-copy is ready for processing.

### Removed

### Migration
