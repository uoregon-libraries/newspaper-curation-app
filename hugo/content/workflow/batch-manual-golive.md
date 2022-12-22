---
title: Batch Manual Go-live Procedure
weight: 60
description: Pushing generated batches to production
---

Once a batch has been approved in staging, all essential files (e.g., no TIFFs)
will be copied to the configured `BATCH_PRODUCTION_PATH` location and then NCA
will mark the batch as ready to go live. At that point:

- Visit the batch management page for the batch for load/purge instructions
- Once the batch is purged from staging and loaded to production (and optionally
  re-loaded on staging from the production location), set it as having gone
  live in the batch management page.

We also have a dark archive process.  We move issues to a dark archive "holding
tank" until we have enough data to warrant a transfer:

- Move batches to the "holding tank" (original batches with the TIFFs, from the
  "ready for ingest" location, e.g., `/mnt/news/outgoing`)
- In the database, set batches' `location` to empty ('')
- When enough batches are in the holding tank, run the script that handles the
  move to the dark archive
- Update the batch's `archived_at` date
- About four weeks after the `archived_at` date, we expect the dark archive is
  safe and backed up

There's a script to help with cleanup: `bin/delete-live-done-issues`, built in
a standard `make` run.  This script will take these four-weeks-plus archived
batches and update their status to `live_done`, indicating they need no more
consideration from NCA.  Then all issues associated with any `live_done` batch
will be removed from the filesystem, and their database records' locations will
be cleared to indicate they are no longer on local storage.  This should be run
regularly to prevent massive disk use, since otherwise all TIFFs, JP2s, PDFs,
and XMLs for all issues will stay on your filesystem indefinitely.
