---
title: Fixing "Stuck" Issues
weight: 35
description: Removing issues from NCA which can't get derivatives generated or have other issues leaving them stuck but invisible to the UI
---

Sometimes a publisher will upload a broken PDF that NCA cannot process. For
smaller organizations, these kinds of problems are easy to prevent just via
careful review. But for larger orgs, it's often infeasible to do this, e.g., if
you have enough publishers that you get hundreds of pages uploaded each week.

When an issue gets stuck, NCA currently has no way to indicate this.  This is
one area where a developer used to have to clean up the filesystem and database
manually.  As of NCA v3.8.0, there is a tool which can handle this in a
significantly less painful way.

## Purging Dead Issues

A normal invocation of `make` creates `bin/purge-dead-issues`. This tool's sole
purpose is to find issues which have a failed job and can no longer move
through NCA's workflow.

Under the hood, this command does the following:

- Finds all issues that are valid candidates for purging.  To be valid, an issue:
  - Is in the "awaiting processing" state
  - Has at least one failed job - as in "failed", which means failed forever,
    not `failed_done`, which indicates a temporary failure which was retried.
  - Is not tied to a batch
  - Has no jobs that are pending or in process
- Ends all jobs that were stuck - this means failed jobs as well as any "on
  hold" jobs that had been waiting for a failed job to finish
- Creates a new job to purge the issue.  This uses the same logic as issues
  that are flagged as having errors and removed from NCA.

Once the tool has been run, you'll have stuck issues in the configured
`ERRORED_ISSUES_PATH` ready for review. Note that depending on the problem, you
may still find yourself needing a developer to dig into the job logs to find
out exactly what went wrong.
