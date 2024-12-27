### Fixed

- Devs: database initialization is a bit better in the docker setup: the
  no-longer-necessary seed file is gone and a new SQL initialization script
  exists to pre-build the DB structure that the current DB migrations create.
  This lets you skip the re-running all the migrations every time you reset the
  database (generally a pain when doing lots of testing).
