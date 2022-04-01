---
title: Replacing Live Issues
weight: 55
description: Fixing batches after they've been pushed into production
---

## Helper Script

We've put together a helper script which can automate a lot of the preparation
steps when a single LCCN needs a lot of issues pulled. It is hacky and
hard-coded at the moment, but it doesn't make any changes to anything, so it's
safe to try out, and even modify to suit other use-cases.

To run: `go run scripts/help-remove-issues.go <lccn> <issue dates filename> <path to NCA dir> <path to live batches>`.

This will find all batches that have the given LCCN and dates. It will "parse"
date ranges that are in the form "YYYY-MM-DD thru YYYY-MM-DD" and then find all
valid issues within that range.

It will then produce a complete list of all commands you'll need to run.

### Preparation

- Create a file full of dates in the form `YYYY-MM-DD`, representing all dates
  which need to be pulled
  - Note that **all** editions on the given date will be removed
- Grab https://github.com/uoregon-libraries/batch-issue-remover and build it.
  Copy the binary to your NCA system, as it is presumably capable of writing to
  your batch dir... otherwise how would you build batches?
- Determine where you'll run this script so you can put all necessary helper
  binaries in place
  - You need `find-issues` (from a standard NCA project build) and
    `remove-issues` (from the above-mentioned batch issue remove repo) under a
    `bin` directory in your NCA dir
    - e.g., if your NCA installation is in `/usr/local/nca`, that's your "path
      to NCA dir", and you'll need to place the binaries in
      `/usr/local/nca/bin`.
- Run the helper, e.g., `go run scripts/help-remove-issues.go sn12345678
  /tmp/issue-dates /usr/local/nca /mnt/livebatches`
  - You probably want to capture the output to a file

### Removal

The output of the helper script has three components:

- Removal / rebatch commands using `remove-issues`. Each of these commands will
  generate a new version of a live batch with the given issue keys removed.
  This is a non-destructive operation, as the live batch isn't changed, and the
  new version could simply be deleted if desired without any effect.
- Purge / load batch commands. These pair up for each new batch generated: you
  purge the live batch, then load the new one which has the bad issues removed.
  These commands are destructive but can still be undone because you can simply
  reverse them if necessary: purge the new version of the batch and then load
  the old one.
- Batch destruction commands. These are very permanent, and will dispose of the
  old version of batches, which you no presumably no longer want living
  alongside the new version.

At UO, we also have a separate component we try to manage, which is a "batch
patch" for our dark archive. For each change to a batch that's been archived,
we craft a README.md with the commands needed, and generally include the entire
codebase of any commands we use in order to ensure reproducibility. This is
manual and error-prone, but necessary to ensure disaster recovery efforts will
not result in loading old/broken batches we've since fixed.

## Post-fix

Once you've ingested the new issues, NCA needs to have its cache completely
rebuilt, since it assumes what's live doesn't change (it does a full rebuild
weekly, but this is often too slow for heavy workflows).

- Stop the NCA services (nca-httpd and nca-workers)
- Delete the NCA cache entirely
  - e.g., if `$ISSUE_CACHE_PATH` is `/tmp`: `rm -rf /tmp/batch-list /tmp/batches /tmp/titles /tmp/finder.cache`
- Start the services up. This will take time as the live site has to be re-scanned
