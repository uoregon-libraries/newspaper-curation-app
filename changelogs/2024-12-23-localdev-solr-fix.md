### Fixed

- `scripts/localdev.sh` no longer loads titles into ONI instances immediately
  after database setup is done. This was sometimes happening faster than ONI
  could initialize the Solr schema, which caused wonderful silent Solr problems
  that would prevent loading batches, but with error messages that didn't
  really explain the root problem well.
