### Added

- A new section of NCA has been added: the "Batch" status page. Here batch
  loaders (generally developers) can see what's waiting for a staging load or
  production load, flag batches as ready for QC, etc. Batch reviewers can use
  this area of the site to flag bad issues and/or mark batches ready for
  loading into production.
- Several new types of background jobs are now supported to help with the new
  batch status page actions and to replace functionality previously in the
  "batch fixer" tool.
- New roles for above: batch reviewer and batch loader
- New setting for designating the URL to a staging server running ONI
  (separately from the production ONI URL)

**Note**: the batch status page is a work in progress. There are a few areas
where we've yet to implement planned features, but holding this any longer
didn't make sense. There are still quite a few operations a dev has to do to
get batches pushed live, for instance. There are also a few areas where one may
still need to refer to the outdated go-live docs, so we've kept those around,
just not as part of the official documentation site. (See "OLD-golive.md" in
the project root)

### Removed

- Explanation of batch manual go-live process has been removed from the
  official documentation site.
- "Batch fixer" command-line tool was removed as it should no longer be
  necessary (or even helpful)

### Migration

- Add a value for the new `STAGING_NEWS_WEBROOT` setting, e.g., `https://oni-staging.example.edu`.
- Shut down NCA entirely, deploy the new version, and run the database
  migrations, e.g., with `goose`:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
