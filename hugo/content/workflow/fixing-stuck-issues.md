---
title: Fixing "Stuck" Issues
weight: 35
description: Removing issues from NCA which can't get derivatives generated or have other issues leaving them stuck but invisible to the UI
---

Sometimes a publisher will upload a broken PDF that NCA cannot process.  There
is a safeguard against these kinds of issues: only queue uploaded issues after
careful review.  But it's often infeasible to do this, especially if you have
enough publishers that you get hundreds of pages uploaded each week.

When an issue gets stuck, NCA currently has no way to indicate this.  This is
one area where a developer used to have to clean up the filesystem and database
manually.  As of NCA v3.8.0, there is a tool which can handle this in a
significantly less painful way.

## Purging Dead Issues

A normal invocation of `make` creates `bin/purge-dead-issues`.  This is a
destructive operation, and you will need to be prepared prior to running it so
that you can decide how best to handle the broken issues.  Please read this
document fully!

When run, `purge-dead-issues` will do a lot of logging to STDERR, print out a
"report" of which issues were purged, and write a `purge.json` file describing
each purged issue in some detail.

By default, `purge-dead-issues` will not actually make any changes.  It scans
the database and reports the issues which would be purged, but it doesn't
actually purge them.  Because the process is exactly the same as a live run,
this allows you to carefully review what will happen without anything
destructive occurring.

When you're ready, run the command with the `--live` flag.

## Technical Details

Under the hood, this command does the following:

- Finds all issues that are valid candidates for purging.  To be valid, an issue:
  - Is in the "awaiting processing" state
  - Has at least one failed job - as in "failed", which means failed forever,
    not `failed_done`, which indicates a temporary failure which was retried.
  - Is not tied to a batch
  - Has no jobs that are pending or in process
- Ends all jobs that were stuck - this means failed jobs as well as any "on
  hold" jobs that had been waiting for a failed jobs to finish
- Creates a new job to purge the issue.  This uses the same logic as issues
  that are flagged as having errors and removed from NCA.

All operations are just database changes, and as such a transaction is able to
wrap the entire command.  A single critical failure of any kind prevents any
changes from being made, ensuring a pretty safe run.

In fact, when `--live` is not specified, the transaction is rolled back right
before the code would normally commit it.  This is why the command is able to
give a complete report as if everything had been run without altering the
application's state in any way.
