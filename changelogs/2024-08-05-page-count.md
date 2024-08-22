### Added

- Various views now include issue and batch page counts

### Migration

- Shut down NCA workers and HTTP daemon. You will have potentially several
  minutes of downtime.
- Run database migrations to add the new page count field and then count pages
  for every issue in the database:
  - `make && ./bin/migrate-database -c ./settings up`
- Restart services.
