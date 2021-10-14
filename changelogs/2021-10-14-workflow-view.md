## vX.Y.Z

Brief description, if necessary

### Added

- Issues now store their most recent curation date so that in the workflow view
  they can be sorted by those which have been waiting the longest for review.
- The "Metadata Review" tab on the workflow page shows a rough wait time (since
  metadata entry) per issue

### Changed

- Both the "Metadata Entry" and "Metadata Review" tabs in the workflow page are
  sorted in a deterministic way (used to be just whatever order the database
  chose to return)
  - Metadata Entry returns issues sorted by LCCN, issue date, and edition -
    this isn't useful so much as it gives us some consistency that we
    previously didn't have.
  - Metadata Review returns issues sorted by when metadata was entered, oldest
    first, so it's easy to tackle issues that have been sitting for a while.

### Migration

- Run the database migrations, e.g., with `goose`:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
