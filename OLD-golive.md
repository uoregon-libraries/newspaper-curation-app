# Batch Manual Go-live Procedure

Once a batch has been approved in staging, the following steps must be taken,
at least for the UO workflow:

- Make sure the batch has a valid `tagmanifest-sha256.txt` file
- Copy the batch (sans TIFFs) to the newspaper batch network store, e.g.:
  ```bash
  # $src is something like:
  #   /mnt/news/outgoing/batch_foo_20180918BasaltVampireTramplingCrabgrass_ver01
  # $dest_batch is something like:
  #   /mnt/production/batch_foo_20180918BasaltVampireTramplingCrabgrass_ver01
  rsync -av flags --delete \
    --exclude="*.tif" --exclude="*.tiff" --exclude="*.TIF" --exclude="*.TIFF" \
    --exclude="*.tar.bz" --exclude="*.tar" \
    $src/ $dest_batch
  ```

- Load the batch into production via the chronam / ONI `load_batch` admin command
- Remove the batch from staging via the chronam / ONI `purge_batch` admin command
  - If your staging system mirrors production data, reload the batch from its live location
- Update the batch in the database so its status is "live" and its
  `went_live_at` date is (relatively) accurate.  The `went_live_at` field is
  technically optional, but can be helpful to track the gap between prepping a
  batch and actually loading it.
  - For example: `UPDATE batches SET status = 'live', went_live_at = NOW() WHERE name = 'BasaltVampireTramplingCrabgrass' AND id = 32 AND status = 'qc_ready'`
- Update the batch's issues in the database to be ignored by setting their `ignored` field to 1
  - If you consistently set the batch status to "live" when you load batches
    into production, this is fairly easy in a single SQL statement:
    - `UPDATE issues SET ignored=1, workflow_step = 'InProduction' WHERE batch_id IN (SELECT id FROM batches WHERE status = 'live')`

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
