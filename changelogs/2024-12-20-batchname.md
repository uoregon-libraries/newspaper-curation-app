### Fixed

- Batch names are now stored in the database when a batch is built, rather than
  computed as needed. This fixes weird datestamp inconsistency when a batch is
  generated on a server with one timezone, and links to staging/production are
  generated in a different timezone. This also prevents us from having massive
  problems if we make a major change to the batch name generation algorithm.

### Migration

- Run database migrations:
  - `make && ./bin/migrate-database -c ./settings up`
