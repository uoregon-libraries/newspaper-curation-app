### Added

- Critical paths in the code, particular the job runner, now have even more
  resiliency! Previously, jobs would auto-retry on failure, but only if the
  database was reachable, because the database is where NCA stores job data.
  Now, in addition to the DB-backed retry, extremely critical paths will
  actually wait and retry when the database is unreachable. It won't wait
  forever, but for handling minor outages or a DB server reboot, this should
  fix a whole slew of very annoying data-cleanup problems.
