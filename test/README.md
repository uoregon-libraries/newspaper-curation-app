# Testing NCA

For a "realistic" test, this directory has a pile of scripts in bash and Go
that enable the creation and manipulation of fake data without having to do a
whole lot of repetitive data entry.

## Prerequisites

To make this work, you will want to do local development with docker
"assistance". This means you have docker running the database, the IIIF server,
and the SFTP server, but your NCA-native apps are compiled locally and run
directly. You can try to follow along using a different approach, but this
guide is meant for that use-case since it's the approach we use for quick dev
work.

You will need some pieces of the local dev script. You can read it and try to
do things manually, but the easiest approach is simply to source the script
into bash: `source <NCA root>/scripts/localdev.sh`.

## Source Data

To start, you need source data. We have [a repository of Oregon data][1] for
this, but it may not be too useful to everybody since you'll have to create
dummy titles and MARC org codes. But it could be useful to look at or as a
starting point if you change some directory names to match your actual titles.

[1]: <https://github.com/uoregon-libraries/nca-test-data>

All data you want to bring into a test NCA instance will live under
`sources/scans` or `sources/sftp`. The `scans` dir would be for issues which
have TIFFs and PDFs, where the TIFF is the record of source and the PDF has the
OCR data. `sftp` is for born-digital issues which only have PDFs.

Naming conventions must be adhered to for the test code to properly put issues into NCA:

- Scanned issues: `sources/scans/<MARC org code>-<LCCN>-<Date><Edition>`.
- SFTPed issues: `sources/sftp/<LCCN>-<Date><Edition>`.

The MARC org code is going to be something like "oru", and is typically three
letters. LCCN is always 10 characters. Date is 8 characters in the form of
`YYYYMMDD`, so January 2nd, 2006 would be `20060102`. Edition is a two-digit
value for the issue's edition in case there were two editions published the
same day. This is almost always `01`.

Examples:

- `sources/sftp/sn83008376-2017011301`: The January 13th, 2017 edition of *The
  Daily Astorian*.
- `sources/scans/oru-sn96088073-1801050801`: The May 8th, 1801 edition of *The
  Bohemia Nugget*, using the MARC org code representing "University of Oregon
  Libraries".

### Want A Subset?

The `test/` directory ignores anything below it beginning with "sources", so if
you want, you could have a huge list of source issues in something like
"sources-all", and just copy in a subset of issues when you are testing a
specific situation. This will make the ingest and curation a lot faster.

## Ingest Sources

The easiest way to blow away the database and get the data started ingesting is
by using `scripts/localdev.sh` in the root of the NCA project. It exposes a
function, `prep_for_testing`, which does the following:

- Deletes everything in the database
- Loads seed data from `docker/mysql/nca-seed-data.sql`:
  - This is a hard-coded filename - it must be *exactly* as written above.
  - This file is *not* provided - you have to export things yourself or write
    your own SQL here. The two critical tables for getting data to work are
    `titles` and `mocs`.
  - Currently this requires you to do local development for the Go side of
    things (because that's how I'm doing it) and use docker for the external
    services (MySQL, IIIF server, and SFTPGo).
- Removes all files from your fake newspaper "network mount"
  (`test/fakemount`).
- Runs `copy-sources.go` to bring all source files into the fake mount.
- Runs the NCA bulk issue queue app to take every born-digital issue and
  prepare it for page renumbering.
  - This is run once per LCCN because of how NCA's bulk issue queue app was
    built. The results are fine, but it's slow and the output may be confusing.
  - Scanned issues are pulled in by a background job already, since we always
    assume in-house scans are already built to spec.

## Run Workers

Run the NCA job runner, e.g., `workers` or `workonce` if you sourced in the
local dev script.

Depending on how many sources you defined, and what type, this could take no
time or it could take a very long time. Scanned images are processed the moment
NCA sees them, which means derivatives have to be generated. SFTPed issues,
however, are simply pre-processed briefly (pages are split and converted to
PDF/a) and then moved to the page review directory.

When all jobs are complete (you may have to look in the `jobs` table manually
for this, e.g., `select * from jobs where status not in ('success',
'on_hold');`), you should see all *valid* issues moved either to the NCA
workflow location (scanned issues) or the page review location (sftp issues).

## Page Review

To simulate a page review pass, the script `rename-page-review.sh` will get the
issues in page review ready for ingest into NCA. It renames to the NCA file
naming spec (0001.pdf, 0002.pdf, etc.) and then generates a manifest file for
each issue (so that you don't have to wait for NCA's server to do that).

To tell NCA these issues were processed a while ago, run `make-older.sh`. This
hacks all manifests to say the issue was last changed four days ago, which will
allow you to queue these issues for processing.

Note that you may want to manually (re)start the job runner. It checks the
filesystem fairly infrequently, so if it's already running, it could take
several minutes for the issues to make their way into NCA.

## Curation and Review

If you're looking to test things that come after metadata entry and/or metadata
review, the `run-workflow.go` script can automate one or both pieces.

To enter metadata:

```bash
go run run-workflow.go -c ../settings --operation curate
```

To review metadata, the command is the same, but the operation is
"review":

```bash
go run run-workflow.go -c ../settings --operation review
```

This script iterates over all issues that are in need of the given operation
and then runs said operation: if you ask for curation, all issues awaiting
metadata entry will have dummy data entered, while asking for review simply
approves all issues awaiting metadata review.

For review, it also queues the job which finalizes the metadata (generating
METS XML). Once those jobs run, issues will be ready for batching via the
standard `queue-batches` command.

## Recipes

If you're looking for some test recipes, we're slowly building some up in the
[recipes subdirectory](./recipes), including runnable scripts which automate
most of the time-consuming testing tasks for specific test cases. These will
never likely work exactly as written for other situations, but they should be a
useful guide to help construct your own tests.
