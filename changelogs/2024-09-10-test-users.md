### Added

- New script for testing, `test/create-test-users.go`, which deletes existing
  users and then creates a new user for each role in NCA

### Changed

- When running `prep_for_testing`, migration-installed users will be destroyed
  and replaced with role-based users. If you tend to test as "admin", nothing
  will appear to change, but the "sysadmin" user will no longer be available.
