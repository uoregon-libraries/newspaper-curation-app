### Changed

- The batch queue command-line app will no longer allow issues that are tied to
  unvalidated titles into a batch. This means that a title *must* have a valid
  LCCN that exists elsewhere (usually Chronicling America or your production
  ONI site).

### Migration

- If you use titles without MARC records in ONI somehow, you'll need to either
  start generating valid MARC records or else manually edit the database in NCA
  to flag titles as having been validated. *This is not recommended*, and
  therefore there will be no instructions for doing so.
