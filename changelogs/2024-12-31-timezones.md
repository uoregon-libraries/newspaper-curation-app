### Fixed

- All MariaDB triggers use UTC time instead of local time
- All times displayed to users are local instead of some being UTC

### Added

- Devs: The integration tests now log the output of `bin/queue-batches` for
  easier debugging.

### Changed

- Devs: the `init.sql` regenerating helper function in `scripts/localdev.sh` is
  now a *destructive* operation. This is necessary to ensure a valid starting
  state for `init.sql`, rather than the situations where it ends up being easy
  to accidentally add test data.
