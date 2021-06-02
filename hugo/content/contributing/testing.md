---
title: Testing
weight: 40
description: Automated and manual testing of NCA
---

## Unit Testing

Running unit tests is easy:

    make test

This compiles all of the code and tests any `*_test.go` files.  Test coverage
is spotty at best right now, but the compile-time checks catch the most common
problems, like typos in variable names.

Contributors: feel free to add more unit tests to improve overall coverage!

## Manual Testing

This is clunky and hacky, but it's what we've got for now.  How it all works:

### Setup

- Get NCA working via docker-compose (see our
  [Development Guide](/contributing/dev-guide); this whole <s>brittle mess</s>
  test suite depends on testing on a docker-enabled system)
- Make sure your docker compose overrides mount `test/fakemount` as
  `/mnt/news`.  The override example is set up to do this, so just copying
  that, as explained in the development guide, will get you up and running.
- Set up titles and MARC org codes in your dockerized NCA instance

Put some test issues into `test/sources/scans` and `test/sources/sftp`:

#### Copying titles manually

- The issues should be exact copies of production issues with all the PDF,
  JP2, and XML files.  For scanned issues, the TIFFs should also be included.
  - The JP2 files are pretty optional, but if you can get them quickly, they
    can be handy for verifying that what NCA produces looks roughly the same
  - The XML files are also optional, but again can be helpful for verifying
    that NCA did things correctly
- Each issue will have a folder name that defines it:
  - SFTP: `LCCN-DateEdition`; e.g., `test/sources/sftp/sn12345678-2001020301`
    would be the February 3, 2001 edition of the title with LCCN `sn12345678`
  - Scans: `OrgCode-LCCN-DateEdition`; e.g.,
    `test/sources/scans/oru-sn12345678-2001020301` would be the February 3,
    2001 edition of the title with LCCN `sn12345678`, and attributed to the
    awardee `oru`.

Example of copying from UO's dark archive:

    cp -r /path/to/newspapers/batch_oru_20160627AggressiveEclair_ver01/data/sn00063621/print/2015022001 \
          ./test/sources/sftp/sn00063621-2015022001

A command like this can get you set up for fake SFTP file processing.  If we
had TIFFs in this issue, the command would look like this:

    cp -r /path/to/newspapers/batch_oru_20160627AggressiveEclair_ver01/data/sn00063621/print/2015022001 \
          ./test/sources/scans/oru-sn00063621-2015022001

#### Pulling external titles

The `pull-issue.sh` script is a good example of grabbing all but the JP2s of
another site's issue and faking it as having been a born-digital upload.  You
may have to tinker with the command some, but it should be easily modified to
copy any live issues you may want to test.

#### Using UO's test issues

Get into the `test` directory and clone our test source issues:

    cd test
    git clone git@github.com:uoregon-libraries/nca-test-data.git sources

Despite the size of the download, this represents very few
useful test cases.  It's more of a way to get started with the app than any
kind of comprehensive set of test issues.  Also note that some of the data is
purposefully incorrect or broken in order to test how NCA responds to it.

In other words, you should probably craft your own test data, but this *is*
available to help get you started if you need it.

### Test

**You need to install the UO gopkg project for this to work**:

    go get -u github.com/uoregon-libraries/gopkg/...

*Note that most shell scripts you'll run need sudo - they assume docker is
controlling your files, which means you need to be root to change them.  The
scripts actually switch ownership back to whatever `whoami` evaluates to.*

***Another Note***: You can manually run the `makemine.sh` script occasionally if you
need to look at the data that's owned by root.  This script is called by the
other scripts and is an encapsulated way to just change ownership quickly.

Once you have the titles and MOCs set up in the front-end, and your `sources`
directory has issues, you're ready to actually use the data:

- Run `reset.sh`.  This will delete all issues, batches, jobs, and job logs
  from the database.  It will then copy (hard-link to avoid disk space bloat)
  the files in `test/sources` into `test/fakemount`.  Assuming the folder names
  are correct in `test/sources`, the layout will be correct in
  `test/fakemount`.
  - `reset.sh` requires the stack to be up and running.  It's exceedingly naive
    in its approach to ensuring the database is in a good "starting" state.  If
    the script fails, make sure to start up your stack first.
- Look at the Uploads section of the NCA web app - you should see whatever
  issues you've put in for testing, and you can queue them for processing
- Queue issues, make sure the "workers" container doesn't throw up
  - If an issue was from sftp (these are born digital issues), they will have
    some preprocessing done and then get moved to the page review area,
    `test/fakemount/page-review`
  - If an issue was scanned, it will have derivatives build and get put into
    the workflow area, `test/fakemount/workflow`
- Fake-process the page-review issues:
  - `rename-page-review.sh` assumes all born-digital issues' pages are in order
    and just names them in the correct format (0001.pdf, 0002.pdf, etc.)
  - `make-older.sh` will fake the files' age so the issues can be processed in
    the app without the "too new" warning.
  - *Wait*.  It takes a few minutes for the workers to scan for page reviews
    (you can watch them via `docker-compose logs -f workers`), and then a few
    more for the web cache to get updated.
- Enter metadata, review metadata, and fire off a batch when ready
  - Queueing a batch through docker:
    - `docker-compose exec workers /usr/local/nca/bin/queue-batches -c ./settings`
  - The batch will end up in `test/fakemount/outgoing`

### Saving State

At any time you can save and restore the application's state via the top-level
`manage` script.  This script has a variety of commands, but `./manage backup`
and `./manage restore` will back up or restore **all files** in the fake mount
as well as all data volumes for NCA, assuming your docker install puts data in
`/var/lib/docker/volumes` and you don't change the project name from the
default of "nca".

This can be very handy for verifying a process such as generating batches,
where you may want to have the same initial state, but see what happens as you
tweak settings (or code).
