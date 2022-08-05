---
title: '"Un-push" Batch From Production'
weight: 70
description: Safely removing generated batches from production
---

Sometimes a batch is messed up enough that it needs to be completely removed,
rebuilt, and reingested, but it's already in production. In the (rare) cases
this happens *and* we haven't already archived all the original files, we can
un-push the batch and requeue the necessary issues.

This procedure helps "un-push" batches, but only when all of the following are true:

- The batch was created by NCA, not a vendor. There's currently no procedure
  for reading a non-NCA batch and putting its issues in the database.
- The batch is live, but hasn't been fully archived yet
  - In theory you could still do this after archival, but the work gets a lot
    more involved and isn't in scope here.
- All issues are still in NCA's database and their files are still in the NCA
  "workflow" location on disk (this is usually true until archival).

This process is awful and you need to know what you're doing, but here's the rough outline:

- If at all possible, turn off NCA's services (workers and http). If you can't
  do this, things could get messy if you have a problem and have to rollback
  database or filesystem changes.
- **Back up your database!** The filesystem may be too much to back up, so it
  can be a pain if you need to fix something, but it's still really helpful to
  at least have a "good DB state".
- Move your batch to the the "ready for ingest" location (e.g.,
  `/mnt/news/outoging`) from wherever it goes to be archived
- In the database, set the batch's `location` to the *full path* to the batch
  you just copied
  - e.g., `/mnt/news/outgoing/batch_foo_20180918BasaltVampireTramplingCrabgrass_ver01`
- "Un-ignore" the issues, but set their state "ReadyForRebatching" rather than
  the typical "ReadyForBatching". This is important to avoid accidentally
  putting these into new batches before you've fixed things.
  - e.g., `UPDATE issues SET ignored=0, workflow_step = 'ReadyForRebatching' WHERE batch_id = ?`
- Do whatever fixes you need to (in our case, altering the `marc_org_code` for a bunch of issues)
- Purge the batch from staging **and** production.
  - This seems scary, but at this point you still have the live batch, the
    archival files, *and* a database backup.
- *Delete* the batch from your live location.
  - This seems scary, but if you're following the [manual go-live](/workflow/batch-manual-golive)
    docs, your live files are just a subset of the archived batch which you
    just copied into the "ready for ingest" location.
- Mark the batch 'deleted':
  - `UPDATE batches SET status = 'deleted' WHERE id = ?`
  - This isn't scary because you made a database backup. Right?
- Remove batch-issue connection for affected issues:
  - `UPDATE issues SET batch_id = 0 WHERE batch_id = ?`
  - Still not scary
- Create a new fixed-issue batch (or batches):
  - `/path/to/nca/bin/queue-batches -c /path/to/nca/settings --redo`
  - This will rebatch *all* issues at the `ReadyForRebatching` workflow step.
    If you only process one problem batch at a time, though, this shouldn't do
    anything but rebuild whatever was busted (meaning you may not need to re-QC
    it, or if you do, it's at least an isolated set of issues).
