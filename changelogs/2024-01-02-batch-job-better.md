### Fixed

- Manual batch loading instructions are now correct for all cases (staging or
  production) due to the changes below.

### Added

### Changed

- Once batches are built, their "always-online" files are immediately synced to
  the live location rather than waiting for QC approval. This improves the
  typical case where a batch is approved and loaded to production, but it also
  simplifies batch processing anyway. Staging and production servers can use a
  unified location for batch loading, and staging won't need a batch purge
  simply to reload it from the final path.
- In our settings example file, we have a clearer explanation of what the two
  batch path values mean.
- Minor changes to some "public" functions. But nobody should use NCA's "public
  API". This repo was built before Go had the "internal" concept to make
  private packages, and hasn't been refactored properly yet. NCA's APIs aren't
  really ever meant to be public.
- Batches are no longer ready for a staging load until *after* their BagIt
  files are generated. This is technically an unnecessary delay, but we cannot
  sync to the production dir until BagIt files are done, and the whole point of
  this unification of batch locations was to avoid having to load from two
  different directories.

### Removed

- The "copy batch for production" job is no longer needed, as we sync files
  immediately on batch build as mentioned above.

### Migration

- Make sure your staging server can read the `BATCH_PRODUCTION_PATH` if it
  couldn't previously. NCA always assumed staging could read both
  `BATCH_PRODUCTION_PATH` and `BATCH_OUTPUT_PATH` anyway, but who knows how
  it's being used in the wild.
