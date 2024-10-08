### Fixed

### Added

### Changed

- NCA now *requires* a Open ONI running on a server alongside an [ONI
  Agent][oni-agent] daemon, reachable by NCA. ONI Agent helps to simplify the
  process of loading and purging batches on the ONI side, drastically
  simplifying the NCA process of managing batches, and eliminating some
  inconsistencies / problems associated with managing batches manually.
  - There are currently *no plans* for manual support of batch loading, due to
    the unnecessary complexity in running automation side-by-side with the
    manual process. For dev use, the docker setup now includes a service to
    fake the automated operations, which could potentially be used in
    production if users didn't mind NCA not actually specifying which
    operations need to be run.

[oni-agent]: <https://github.com/open-oni/oni-agent>

### Removed

- All manual flagging of batch loading and purging has been removed, as have
  the instructions for these pieces.

### Migration

- Get all batches out of NCA's "manual" workflow states prior to upgrading.
  Behavior will be undefined, but almost certainly unpleasant, if there are
  batches in some of the states that no longer exist, such as batches waiting
  for a staging purge.
  - Safe states: `live`, `live_done`, `live_archived`, `qc_ready`,
    `qc_flagging`, or `deleted`
  - Unsafe: `pending`, `staging_ready`, or `passed_qc`
