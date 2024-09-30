### Changed

- Most areas of NCA that used the term "purge", when referring to something
  other than the purging of a batch from ONI, have changed to some other term.
  This requires a migration (see notes below) to keep the database meaningful.

### Migration

- Shut down NCA workers and HTTP daemon
- Run database migrations:
  - `make && ./bin/migrate-database -c ./settings up`
- Restart services
