### Fixed

- Internal refactors which affect the "public" APIs. But don't use these. I
  just haven't gotten around to moving everything to a proper "internal"
  subdirectory.

### Added

- NCA's "Titles" view now has a link to upload MARC XML. If you do an upload,
  two things happen:
  - NCA will create or update any record with a matching LCCN with the basic
    MARC data it cares about: LCCN, name, location, and some internal fields
    like the fact that it considers the title's LCCN to be validated.
  - The ONI agents (staging and production) will be given a command to load the
    MARC record into ONI. This will mean *no more batch failures* due to a
    missing title.
- You can use the ONI Agent test binary to load titles into ONI if you need to
  do some testing. Otherwise just use NCA so all the pieces stay consistent.

### Changed

- A lot of the integrated testing (automated "manual" testing in the `test`
  directory) has been simplified and improved. You no longer need to bring your
  own seed SQL if you use our test repository. You can create MARC records that
  NCA will auto-load for your own test data. If you test this way, re-read the
  README in its entirety!

### Migration

- If you had a custom nca-seed-data.sql, you should rename it, and you'll have
  to abandon it or manually load it when needed. This part of the integration
  suite is no longer done for you to make it easier for a typical developer to
  just dive into the tests.
