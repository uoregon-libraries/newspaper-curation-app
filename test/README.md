Testing NCA
===

This is clunky and hacky, but it's what we've got for now.  How it all works:

Setup
---

- Get NCA working via docker-compose; this whole <s>brittle mess</s> test suite
  depends on testing on a docker-enabled system
- Make sure your docker compose overrides mount `test/fakemount` as
  `/mnt/news`.  The override example is set up to do this, so just copying that
  will get you up and running faster.
- Set up titles and MARC org codes in your dockerized NCA instance
- Put some test issues into `test/sources/scans` and `test/sources/sftp`
  - The issues should be exact copies of production issues with all the PDF,
    JP2, and XML files
  - Each issue will have a folder name that defines it:
    - SFTP: `LCCN-DateEdition`; e.g., `test/sources/sftp/sn12345678-2001020301`
      would be the February 3, 2001 edition of the title with LCCN `sn12345678`
    - Scans: `OrgCode-LCCN-DateEdition`; e.g.,
      `test/sources/scans/oru-sn12345678-2001020301` would be the February 3,
      2001 edition of the title with LCCN `sn12345678`, and attributed to the
      awardee `oru`.

Test
---

*Note that most shell scripts you'll run need sudo - they assume docker is
controlling your files, which means you need to be root to change them.  The
scripts actually switch ownership back to whatever `whoami` evaluates to.*

- Run `reset.sh`.  This will delete all issues, batches, jobs, and job logs
  from the database.  It will then copy (hard-link to avoid disk space bloat)
  the files in `test/sources` into `test/fakemount`.  Assuming the folder names
  are correct in `test/sources`, the layout will be correct in
  `test/fakemount`.
- Look at the Uploads section of the NCA web app - you should see whatever
  issues you've put in for testing, and you can queue them for processing
- Queue issues, make sure the "workers" container doesn't throw up
  - If an issue was from sftp (these are born digital issues), they will have
    some preprocessing done and then get moved to the page review area,
    `test/fakemount/page-review`
  - If an issue was scanned, it will have derivatives build and get put into
    the workflow area, `test/fakemount/workflow`
- Fake-process the page-review issues:
  - `rename-page-review.sh` spits out a bunch of bash commands which will
    rename the page-review issues.  You can pipe the output directly into bash
    or run commands more selectively in order to choose specific issues to
    rename.
  - `make-older.sh` will fake the files' age so the issues can be processed in
    the app without the "too new" warning.
- Enter metadata, review metadata, and fire off a batch when ready
  - Queueing a batch through docker:
    - `docker-compose exec workers /usr/local/nca/bin/queue-batches -c ./settings`
  - The batch will end up in `test/fakemount/outgoing`

Saving State
---

At any time you can save your state via `backup-state.sh`.  This creates a tar
of *all files* in the fake mount as well as an SQL export of all data in the
dockerized database (assuming your docker install puts data in
`/var/lib/docker/volumes` and you don't change the project name from the
default of "nca").

This *must be run as root* in order to access the mysql files.

You can run `restore-state.sh` to put all the files back into `fakemount`, and
restore the database.

This can be very handy for verifying a process such as generating batches,
where you may want to have the same initial state, but see what happens as you
tweak settings (or code).
