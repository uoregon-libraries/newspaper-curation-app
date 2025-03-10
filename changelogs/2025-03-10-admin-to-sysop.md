### Added

- New role, "Site Manager", to represent a user who isn't a dev/ops person, but
  has most access in NCA.
- Dev: unit testing for privileges and roles!

### Changed

- The role "admin" is replaced with "sysop", as in "system operator"
  (old-school term). This should be both weird enough and also clear enough to
  state this is somebody who has absolute power, not to be confused with a
  non-technical admin like the project manager.
  - As part of this, a lot of application text has been clarified or updated,
    as has our documentation.
- The database state created by NCA now includes a user with the login "sysop"
  rather than "admin", and the previous "admin" and "sysadmin" users, if they
  are in your system, will be deactivated.
  - Dev: this also changes the seed data! You'll have to adjust your URL
    parameter from `debuguser=admin` to `debuguser=sysop`.
- Dev: privileges and roles have been massively refactored to make future
  changes a lot less cumbersome.

### Removed

- Admins (now SysOps) no longer have access to "hack" a batch URL to view
  deleted or in-process batches. This was undocumented, so shouldn't affect
  anybody greatly, and probably shouldn't have been added in the first place.

### Migration

- Migrate the database:
  - **Important**: if you're using "sysadmin" or "admin" today, (a) *stop doing
    this and create real users*, but (b) if you really need generic
    system-built users, **those two will be deactivated** after migrating. You
    will need to change your processes to rely instead on the "sysop" user.
  - `make && ./bin/migrate-database -c ./settings up`
- Consider downgrading people who have "sysop" to the new "Site Manager" role.
  This role has access to most things, but won't be able to perform some
  actions that are dangerous. Moving forward, sysops will get more and more
  dangerous privileges, while the site admins will only get privileges that
  are safe for a non-dev user.
