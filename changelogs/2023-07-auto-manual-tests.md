### Added

- New flag for job runner to auto-exit when there are no more jobs to run. This
  isn't really a production feature, since you want jobs to always be checked
  and run in production, but rather a way to help script tests so there are
  fewer interactive steps when running them.

### Changed

- General test is now a script with minimal interactive pieces instead of a
  document describing the steps you'd have to take.
