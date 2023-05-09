# NCA Changelog

All notable changes to NCA will be documented in this file.

Starting from NCA v2.10.0, a changelog shall be kept based loosely on the [Keep
a Changelog](https://keepachangelog.com/en/1.0.0/) format.

Note that this has historically been an internal project for UO, and some notes
have been confusing or even incomplete in the past.  We try hard to avoid this,
but as usual, ye be warned.  However, if you're using NCA and find notes that
don't help or seem wrong, please do not hesitate to
[file an issue](https://github.com/uoregon-libraries/newspaper-curation-app/issues)!

Starting from NCA v3.9.0, changes are kept in a separate `changelogs` directory
until a branch is merged into main in order to minimize the difficulty merging
changes when fixes and long-running features happen in tandem.  This *will* add
ugliness and noise to git work, but it's better than the merge conflicts and
error-prone resolutions we've been dealing with.

<!-- Template

## vX.Y.Z

Brief description, if necessary

### Fixed

### Added

### Changed

### Removed

### Migration
-->

## v4.0.0 (unreleased)

The NCA 4 release fixes and improves a lot of areas, but the biggest change is
our SFTPGo integration and documentation! This means you don't have to try and
figure out how to get publishers' content into NCA if you just stand up an
SFTPGo instance and "connect" NCA to it.

It's mentioned below, but to upgrade to 4.0, you should first read the
[SFTPGo documentation](https://uoregon-libraries.github.io/newspaper-curation-app/setup/sftpgo-integration/).

### Fixed

- `.manifest` files are now ignored in the UI, as they are "internal" to NCA
  and not something curators need (or want) to see
- Flagging issues from the batch status view will no longer result in a crash
  of the workers daemon. OUCH.
- Batches that have issues removed will no longer "stall" in the job queue
  (`Batch.AbortIssueFlagging` allows pending batches now in addition to those
  flagged as needing QC)
- On some systems, `.manifest` files were seen by NCA as constantly changing,
  which prevented issues from getting metadata entered or even entering the
  workflow at all. This should no longer be a problem.
- Two horrible panic calls from deep within NCA's bowels have been replaced
  with error returns, which should dramatically reduce the risk of a runtime
  crash. These were already rare, but now *should* be nonexistent.
- Invalid unicode characters (anything defined as a control rune, private-use
  rune, or "surrogate" rune) are stripped from the output `pdftotext` gives us
  just prior to generating ALTO XML. This prevents MySQL and MariaDB errors
  when ingesting into ONI.

### Added

- [SFTPGo](https://github.com/drakkan/sftpgo) is now integrated with NCA for
  managing titles' uploads.
- Batch Management:
  - Various instructions and status-setting buttons have been added to the batch
    management page for batch loaders
  - Instructions for batch loaders' manual tasks now have a "copy" button, which
    should make manual tasks a bit easier
  - Batches which were once on staging now have to be marked as purged from
    staging before they can move to other statuses (e.g., loading to production)
  - A new setting, `BATCH_PRODUCTION_PATH`, has been introduced. Set this to the
    location NCA should copy your batches when they're ready for being ingested
    into production.
    - On QC approval, batches are automatically synced to the location specified by
      the new setting (`BATCH_PRODUCTION_PATH`).
  - A new setting, `BATCH_ARCHIVE_PATH`, has been introduced. Set this to the
    location NCA should move your batches after they're live.
    - Once a batch is marked live, NCA will kick off a job to move all files
      out of NCA and into this location.
  - Batch loaders can now mark issues as live, which moves them to the
    aforementioned location, and as archived, which allows
    `bin/delete-live-done-issues` to remove their workflow files (after a delay).
- Error handling:
  - Many functions were returning errors which were silently being ignored, and
    are now properly being looked at (or at least explicitly ignored where the
    error truly didn't matter)
  - Hard crashes should no longer be possible in the web server (the `nca-http`
    service) even if we have missed some error handling or there's some crazy
    crash we haven't yet found.
  - Hard crashes from workers (the `nca-workers` service) should no longer be
    possible, though there are still some paths which can't be made 100%
    foolproof. Even so, if there *are* areas that can still crash, (a) we will
    fix them when we find them, and (b) they should be ridiculously rare.
- General:
  - The `manage` script restarts key services after shutting them down
  - New document added to the "test" directory to help explain how to create data
    and use helper scripts when manually testing NCA in a real-world-like setting
  - New test script to enter and review dummy metadata for quicker testing
  - New documentation created to help devs create new configuration settings.

### Changed

- Users with the role "Title Manager" can now edit all aspects of a title,
  including SFTP data. Since we no longer store plaintext passwords, there's
  no reason to do the weird half-editing setup we had before where only admins
  could edit the SFTP stuff.
- More intuitive redirects from batch management pages
- In the batch management handlers, inability to load a batch due to a database
  failure now logs as "critical", not just "error".
- All invocations of `docker-compose` are now `docker compose` (note the space
  instead of the hyphen) so people installing docker and compose in the past
  year or two aren't confused (or laughing at how outdated NCA is).
- For devs:
  - We replaced `golint` with `revive`. `golint` is deprecated, and `revive`
    offers a lot more rules to catch potential errors and/or style issues
  - We replaced "interface{}" with "any" for readability now that we're well
    past Go 1.18
- Massive overhaul of workflow and batch management documentation to match the
  new processes

### Removed

- SFTP Password and SFTP directory fields are no longer stored in NCA, as
  neither of these fields had any way to tie to a backend SFTP daemon, and got
  out of sync too easily
- "Failed QC" has been removed as a batch status, as it is no longer in use
- Various bits of dead code have been removed. Some were technically part of
  the public API, but NCA's code isn't meant to be imported elsewhere, so if
  this breaks anything I will be absolutely FLABBERGASTED. Yeah, I said
  "flabbergasted". Deal with it.
- Most manual testing documentation in hugo has been removed in favor of the
  test directory being more self-documenting. We chose to gut the hugo side
  because our approach to manual testing is somewhat UO-specific, and a lot
  more internal than what we want in our public docs - only core devs will care
  about this stuff.

### Migration

- Database migration, e.g.:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
- Set up SFTPGo if desired. The docs cover
  [SFTPGo integration](https://uoregon-libraries.github.io/newspaper-curation-app/setup/sftpgo-integration/),
  including how to *not* integrate with SFTPGo.
  - Note that if you don't integrate, but had been relying on the SFTP fields,
    you will lose this functionality. Due to maintenance difficulties and
    complexity in trying to wrangle conditional use of this data, NCA will no
    longer manage those fields or even display them.
  - If you switch from a traditional sftp daemon to sftpgo, there will be a
    service disruption publishers need to be made aware of. Plan for an entire
    day just to be on the safe side, as you'll need to shut down your existing
    service, copy anything over to sftpgo's location (or make sure you get
    through the queue of what's in your sftp server first), stand up sftpgo,
    connect NCA and then test uploading, downloading, etc. to be sure you've
    got all your OS-level permissions set up properly.
- Get every pending batch out of NCA and into your production systems,
  otherwise batches might get processed incorrectly.
- Do not update if you have batches in the `failed_qc` status. Get batches out
  of this status (e.g., by running the now-defunct batch fixer command-line
  tool), because it is no longer valid.
- Make sure you have "docker compose" installed as a plugin.

## v3.13.1

Code-named "batch status page hotfixes". As promised, there were indeed several
very exciting bugs with v3.13.0. This release aims to fix the bugs we've
managed to catch so far after a bit of live use.

### Fixed

- Changelog no longer claims things were removed which in fact were not. Yeah,
  even the changelog had bugs.
- Batch status is properly set to "Ready for staging" after a batch is built
  instead of claiming to already be ready for QC (which means it's *on*
  staging).

### Added

- Batch reviewer and batch loader roles can now be assigned in the UI!

### Removed

- Batch fixer CLI docs are properly removed. The page was removed from our Hugo
  docs, but the resulting HTML was accidentally not properly updated.

### Migration

If you generated any batches, their status will claim they're ready for QC. You
should load the batches onto staging such that that status is correct. Once
this fix is deployed, newly generated batches will correctly tell you they need
to be loaded onto staging.

You can also get around this by manually changing the batches' statuses to
`staging_ready` in the database, but this can be riskier if you don't know
precisely which batches in the database need to be changed.

The SQL **might** look like the below example. Tweak to suit your situation!

```sql
UPDATE batches SET status = 'staging_ready' WHERE status = 'qc_ready';
```

## v3.13.0

Batch partial-automation mega-update! Bug fixes, too.

### Fixed

- Issues will no longer be able to move into NCA while still being uploaded
  (see "Changed" section for details)
- Fixed test scripts often printing out an irrelevant error: `chmod: cannot access './fakemount/workflow/*': No such file or directory`

### Added

- Batch Status Pages:
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
  - **Note**: this part of NCA is a work in progress. There are a few areas
    where we've yet to implement planned features, but holding this any longer
    didn't make sense. There are still quite a few operations a dev has to do
    to get batches pushed live, for instance.
- New command flag for `queue-batches` to only create batches for issues which
  need a "redo". This only works when a manual fix is done in the database
  (described in the documentation), so it is rarely going to be useful, but it
  will help when a batch just went live and was found to need an immediate fix.

### Changed

- Major change to how an issue's "last modified" date is determined. Instead of
  relying on the files' creation/modification times, we now generate a manifest
  file that tells us what the files' sizes and last modified dates are at a
  given point in time. This will make NCA slower when scanning issues, but some
  filesystem copy operations don't seem to properly tell us when the file was
  first copied, instead reporting the file's original creation time. The new
  algorithm will instead let us know the first time NCA sees a file *in
  addition* to when a file changes.

### Removed

- "Batch fixer" command-line tool was removed as it should no longer be
  necessary (or even helpful)

### Migration

- Add a value for the new `STAGING_NEWS_WEBROOT` setting, e.g., `https://oni-staging.example.edu`.
- Shut down NCA entirely, deploy the new version, and run the database
  migrations, e.g., with `goose`:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`

## v3.12.1

A better new-setup experience!

### Fixed

- Various dev- and docker-specific setup scripts and docs have been fixed to
  address shortcomings. These only affected new users, making them not obvious
  to me. Thanks to @michaelmherrera for taking NCA for a test drive and
  uncovering some old cruft!

## v3.12.0

Various fixes, far better in-process dupe-checking

### Fixed

- Live batch issue removal helper script no longer hard-codes the LCCN
- CircleCI test integration works again
- Action logs now report batch name properly
- Dupe checking against issues in NCA's database is now real-time. This fixes
  the very rare (but very horrible to fix as we just learned) situation where
  two issues have the same metadata entered at roughly the same time, and the
  curators receive no warnings about the duplication.

### Added

- New documentation for removing issues from a live batch

### Changed

- Batch fixer's wording makes it clearer that pulled issues will end up in the
  workflow, not removed from NCA
- Various low-level code changes to improve error handling and issue management

## v3.11.4

Miscellaneous improvements; major data problem fixed

### Fixed

- Audit logs no longer store raw binary data instead of the proper text value

### Added

- Slightly better tooling for replacing issues in live batches; documentation
  is still needed here, though

### Migration

- Shut down NCA entirely, deploy the new version, and run the database
  migrations, e.g., with `goose`:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`

## v3.11.3

Hotfix

### Fixed

- The "remove an errored issue from NCA" page no longer fails to load due to
  errors in how I handled loading the issue's errors. So meta.

## v3.11.2

Hotfix

### Fixed

- Partial pages will no longer render - if a page fails to render, it will be *very* clear that something went wrong
- Uploaded issues with file errors can now be seen instead of crashing mid-render

## v3.11.1

Hotfix

### Fixed

- The application no longer crashes when viewing issues flagged as having
  unfixable errors

## v3.11.0

Audit logs and workflow improvements

### Fixed

- Audit logs can now be filtered without having to choose a single individual.

### Added

- New filtering options for audit logs to see grouped logs by general type.
- Some basic info/help/background text is now shown after the audit log block
  to hopefully help explain what exactly the purpose is for these logs.
- Issues now store their most recent curation date so that in the workflow view
  they can be sorted by those which have been waiting the longest for review.
- The "Metadata Review" tab on the workflow page shows a rough wait time (since
  metadata entry) per issue

### Changed

- Audit log types are now in a controlled list rather than just any text, which
  should make the audit logs more meaningful as new stuff needs to be audited.
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

## v3.10.1

Hotfix for workflow filters

### Fixed

- Curators no longer see their own issues in the list of issues awaiting
  metadata approval unless they have the privileges to claim and approve those
  issues. Now that one can only see 100 issues at a time, this is critical for
  workflows with few people but hundreds of issues.

## v3.10.0

Admin Mega-Release: workflow UI improvements, audit log view / download

### Fixed

- In the workflow page, you can now properly click on a tab's badge (the
  roundish dark circle with a number telling you how many issues are in the
  given tab panel) to activate the tab.
- Workflow tab selection no longer adds countless entries to your browser
  history. (i.e., you can use your browser's "back" button to go to the page
  you were at prior to the workflow page instead of having to hit "back" a
  billion times)
- Workflow tabs now manipulate the query string of the URL rather than the
  fragment. This is mostly a "best practice" sort of fix, but it does mean the
  "skip to main content" link now functions properly in the workflow pages,
  where it used to kick you back to the "desk" tab.
- Workflow tabs are HTML headings, which should be an improvement for users of
  tools which let you browse and navigate to headings (e.g., screen readers).

### Added

- New app section for viewing and downloading audit logs (for admins only)
- Workflow UI now has filters to only show issues with a particular LCCN and/or
  MARC Org Code.

### Changed

- Refactor to how issues are pulled from the database, to enable other work
- Workflow lazy-loading: the workflow pages should generally look the same as
  previously, but they no longer pre-fetch gobs of data:
  - A maximum of 100 issues is presented on a tab now
  - Issues are loaded for a tab when the tab is selected rather than all tabs'
    issues being loaded at once

## v3.9.0

Issue error overhaul

### Fixed

- CircleCI builds succeed again (updated Go version in CircleCI config)

### Changed

- Updated to latest Go crypto package
- Errors associated with an issue's metadata are now treated very differently
  than before.  Some are warnings rather than critical errors and can be
  ignored if necessary.
  - Users must explicitly skip warnings on metadata entry if they feel the
    warnings are not a concern
  - The bulk issue queue will fail on any warnings to prevent things like dupes
    from getting into the system.  This change may be re-evaluated later, as it
    could cause an unnecessary burden for issues which need to get queued
    regardless, and where these warnings can be managed in NCA anyway.

### Migration

The NCA disk cache will have to be replaced, because some data structures have
changed in an incompatible way.  There's the easy way and the hard way to do
this.  The easy way requires more downtime:

- Shut off NCA's services
- Delete the cache files as specified by the setting `ISSUE_CACHE_PATH`, e.g.:
  `rm -rf /var/local/news/nca/cache/*`
- Deploy NCA and restart.  The cache rebuild will keep NCA from running for
  several minutes.

Alternatively, you can rebuild the cache outside NCA:

- Build NCA and make sure you have `bin/make-cache`
- Copy your production settings file somewhere else, and edit
  `ISSUE_CACHE_PATH` to point to a temporary location.
- Run `bin/make-cache -c <temporary settings file>`
- Shut down NCA
- Delete the existing cache files and replace them with the ones you just
  generated
- Deploy NCA and restart

## v3.8.0

New tool for purging dead issues, various fixes

### Fixed

- `make format` works again
- Fixed rare edge case with batch jobs being loaded as if they were issue jobs
- Issues which are removed from NCA (via the in-app "report an error"
  functionality or the new `purge-dead-issues` tool) now report *all* actions
  in `actions.txt`, not just those of interest to curators.

### Added

- New tool for getting "stuck" issues out of the workflow (e.g., issues that
  failed to have pages split or derivatives generated due to broken PDFs)
- More internal issue actions logged: when adding an issue to a batch or
  removing one from a batch

### Changed

- Documentation is now housed in `docs/`, and served up via github pages
  (https://uoregon-libraries.github.io/newspaper-curation-app/) instead of
  using the github wiki.

## v3.7.2

Local dev fixes

### Fixed

- Docker builds mostly work again
  - You currently cannot use a built docker image as a standalone entity; you
    have to use docker-compose and mount in local binaries.  This will get a
    github issue.
- Builds rely on goimports instead of gofmt
  - This fixes local development when using versioned go (i.e., when you pull
    down a new version via `go get golang.org/dl/go1.X.Y`)

## v3.7.1

Deploys work again

### Fixed

- Deploy script, broken in v3.7.0, is fixed

## v3.7.0

Backend job fixes and better action log on issues

### Fixed

- Dead jobs which are requeued now properly reset their retry count
- Timestamps should now always be converted to local time instead of sometimes
  being UTC (without even bothering to say so)

### Added

- Metadata approval is now added to the action log
- More internal system events are logged to issue actions so users can better
  see what's happened to an issue beyond just who entered metadata
  - Metadata curation and review continue to only display "relevant" actions to
    avoid losing comments to things like users claiming/unclaiming issues
- Full version information is now embedded into NCA at compile-time based on
  the output of `git describe`

## v3.6.0

Minor improvements, mostly related to to the batch-fixer tool

### Fixed

- The system user can no longer claim issues.  This was an edge case that
  rarely occurred, and in a worst-case scenario slowed things down until the
  workflow owner expired, but it was still a nuisance.
- More consistency and better error handling in the batch-fixer tool when
  rejecting or removing issues from a batch

### Added

- batch-fixer has a new command for rejecting/erroring all issues
  simultaneously.  This should be extremely rarely needed, but it can help with
  weird problems that necessitate an entire batch's issues being sent back or
  deleted from NCA.

## v3.5.1

Minor performance improvements

### Fixed

- Large datasets will now be much faster, thanks to database indices (oops).
  Most operations won't change noticeably, but direct database queries on large
  tables will improve a lot, such as `SELECT * FROM job_logs where job_id = ?`

### Migration

- Make sure you run the latest migrations

## v3.5.0

Reduced barriers for uploaded scans and more consisten in-workflow filenames

### Changed

- Uploaded scanned files have no restrictions on their filename prefixes
  anymore, so long as the prefixes match between TIFF and PDF files
- Page files (*.pdf, *.tif, *.jp2, and *.xml [for alto, not mets]) have
  consistent names again after they move into the internal workflow
- Minor improvement to how `scripts/localdev.sh` launches workers

### Migration

- Empty all jobs in production before deploying this, otherwise there's a
  chance of errors when the jobs start up again.  The chance is small (you'd
  need to have unusual filenames in an issue that is awaiting derivative
  processing), but it's easier to just avoid the situation.

## v3.4.0

### Fixed

- Docker setup is more up-to-date, and the IIIF images should be more consistently working
- Better explanation of how localdev.sh works when using it
- File handling is even *more* robust when really odd edge cases happen, such
  as losing network file mounts or running out of disk space
- Better error reporting when building the issue cache fails

### Added

- The batch fixer tool can now be used to forcibly regenerate derivatives for
  issues, bypassing the normal "if a derivative exists, skip it" logic that
  usually saves time, but sometimes causes problems fixing a bad file
  - This paves the way to expose rerunning of derivatives on the web app if it
    proves necessary to do this prior to an issue being in a batch

### Changed

- The batch fixer now displays issues' workflow steps in the "info" command
- The batch fixer now auto-loads an issue after a "search" command if there is
  exactly one result and the batch is in the "failed QC" status

## v3.3.1

Minor fix to issue visibility in Workflow

### Fixed

- Issues in the workflow tab will not be visibile if the current user cannot
  claim them.  This is especially important for curators who are also
  reviewers, as they cannot review issues they curated, but previously those
  were showing in the list, leading to a lot of errors when trying to claim an
  issue for review after having just finished a large batch of curation.

## v3.3.0

Meaningless version bump because that was forgotten in v3.2.0

## v3.2.0

Bye to old tools, hello to new tools, and small fixes

### Fixed

- Removed "not implemented" from a page that is most definitely implemented
- Some wiki pages were rewritten to match reality more closely

### Added

- New command-line tool to remove issues from the "page review" location safely
- Optional local-dev scripts for advanced users

### Changed

- There's no longer a "develop" branch, as the project "team" isn't big enough
  to warrant a complex branching setup

### Removed

- Several minor and unused binaries were removed to reduce maintenance and
  clean up the compilation process a bit:
  - `makejp2`, which was sort of a one-off to test that the JP2 transforms worked
  - `pdf-to-alto-xml`, a standalone tranform runner for making Alto XML out of a PDF file
  - `print-live-lccns`, basically a weird one-off for some validation which
    probably never should have gotten into this repo
  - `report-errors`, which read the issue cache to report all known errors

## v3.1.0

Error issue removal implemented

### Fixed

- Minor tweaks to incorrect or confusing error messages

### Added

- Issues with unfixable errors that cannot be pushed back into NCA can now be
  moved to an external location designated in the new setting,
  `ERRORED_ISSUES_PATH`.
- Reimplementation of the "force-rerun" action for the run-jobs tool.  This
  time it's built in such a way that you can't accidentally run it - you have
  to look at the code to understand how to make it go, and if you aren't
  comfortable doing that, there's a good chance you shouldn't be using it.

### Migration

- Make sure you set `ERRORED_ISSUES_PATH`, then just stop the old server and
  deploy the new one.

## v3.0.0

Language changes: as much as possible, *all* code and settings no longer refer
to "master" assets, e.g., master PDFs, master backups, etc.  No functional
changes have been made.  The choice to bump versions to 3.0.0 is due to the
settings changes not being backward-compatible.

### Migration

- Shut down your NCA web server
- Wait for your NCA workers to complete *all* jobs, then shut the workers down
- Remove the NCA "finder" cache, e.g., `rm /tmp/nca/finder.cache`
- Update your settings file:
  - `MASTER_PDF_UPLOAD_PATH` is now `PDF_UPLOAD_PATH`
  - `MASTER_SCAN_UPLOAD_PATH` is now `SCAN_UPLOAD_PATH`
  - `MASTER_PDF_BACKUP_PATH` is now `ORIGINAL_PDF_BACKUP_PATH`
- Start NCA workers and web daemon

## v2.14.0

Major workflow improvements and accessibility fixes.  Minor refactoring.

### Fixed

- Various accessibility fixes have been implemented
  - Skip link added before nav
  - Workflow tabs are now saved in the URL
  - Tabs don't trap alt-back or alt-forward anymore
  - Various elements' roles have been fixed to be more semantically correct
  - Submit buttons always have correct and non-empty accessible text
  - "Help blocks" are now associated to their form fields properly
- Permissions checks should be more consistent in the workflow area

### Added

- New errored issues functionality
  - Available to admins and issue managers (see below)
  - Shows all issues which have been reported as having errors
  - Allows returning errored issues back to NCA in cases where an issue isn't
    broken or is good enough to go through despite errors
  - Functionality is planned for removing an errored issue entirely
- New role, "issue manager", who is able to curate, review, "self-review"
  (review issues they curated), and process errored issues
- Workflow tabs now display the number of issues in each tab so users don't
  have to navigate to a tab just to see there's nothing there

### Changed

- Various workflowhandler refactoring has been done:
  - Major changes to the permissions checks so the HTML/UI always does the same
    thing the underlying code does
  - The "Unable to search for issues" error is more consistent now
  - All error reporting and processing of reported issues is in a single file
    for better organization
  - All routing rules live in a separate file
- The docker setup now uses the RAIS 4 Alpine release

## v2.13.2

### Fixed

- Deploy script works again

## v2.13.1

### Fixed

- "Unfixable" error reports are now actions instead of a one-off field,
  allowing for a better history of what happened to an issue

### Removed

- Error issue mover is no more.  A better option is on the way.

### Migration

- Migrate the database, e.g, with `goose`:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
- Take the server down, delete the NCA cache, e.g., `rm /tmp/nca/finder.cache`, and restart

## v2.13.0

### Fixed

- Saving users works again.  Oops.
- Extremely rare edge-case database problems no longer cause job runner to crash

### Added

- New "scanner scanner" in the job runner, watching for scanned issues which
  are ready to be moved into NCA's workflow

### Changed

- Job runner stderr output now logs at "Info" level by default, with an option
  to log at "Debug" level.  Database logging is still always "Debug".

## v2.12.1

### Fixed

- Deploy script works with the version of goose we recommend using.  Oops.

### Changed

- Exponential backoff of failed jobs is now smarter to give failures more time
  to resolve before the first retry
- Invalid filenames are now checked more carefully before issues can get into
  NCA, to avoid dev-only manual filesystem fixing

## v2.12.0

User persistence!

Better handling of filenames!

Issue comments!!!!!!11!!1!1!one!!!1

### Fixed

- Users are now deactivated rather than deleted, which fixes a potential crash
  when viewing rejection notes or passing an issue back to the original
  curator's desk.  Honestly I'm not sure how this never crashed before.

### Added

- Issues can now have comments, and those comments are visible so long as the
  issue is part of NCA's workflow.  They're normally optional, but are required
  when rejecting an issue from the metadata queue (comments replace the
  previous one-off rejection note)
- New top-level `manage` script to simplify development and testing

### Changed

- Filenames for scanned and renamed PDF pages no longer have to be exactly four
  digits (e.g., `0001.pdf`).  As long as they're purely numeric, they're
  considered usable.  This fixes a long-standing problem where pages could be
  accidentally renamed to be three digits, and the process to get them fixed is
  annoying and confusing to the end user.
- Updated `uoregon-libraries/gopkg` to v0.15.0 to get a numeric file sort into NCA

### Removed

- Some dead / obsolete things have been dropped:
  - Removed a dead database table we haven't used in years
  - Removed unused `db/dbconf-example.yml` (this isn't necessary for the new
    fork of `goose` we've been recommending ... for a long time)
  - Removed dead "helper" SQL for identifying issues that can be manually
    deleted from the filesystem (now that there's a tool to handle this, the
    helper sql was just confusing)

### Migration

- Migrate the database, e.g, with `goose`: `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
- Delete the NCA cache, e.g., `rm /tmp/nca/* -rf` - this may not be necessary,
  but with how much the data has changed and how fast NCA is to regenerate it
  (under ten minutes when reading our 1.3 million live pages), it's worth a
  little time for a bit of extra safety.

## v2.11.4

Multi-awardee titles and some fixes

### Fixed

- Updated gopkg to v0.14.1, which should further reduce permissions issues for
  copied files/dirs
- Scanned titles in more than one MARC Org Code directory will now be visible
  and can be queued via the NCA UI
- Buggy page number labeling in templates is fixed
- Batch queueing now shows a proper error when it fails
- Widened a DB column which was causing jobs to fail to be created (very rarely)

### Added

- `delete-live-done-issues` is now deployed to production

### Migration

- Migrate the database, e.g, with `goose`: `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`

## v2.11.3

"Thumbs.db still sucks.  So do permissions."

### Fixed

- Page-review mover also ignores Thumbs.db
- Page-split job gives read/execute permissions to the world on its directory
  to hopefully finally really really fix permissions problems

## v2.11.2

"Thumbs.db sucks"

### Fixed
- File validation ignores Thumbs.db
- File cleaner removes Thumbs.db

## v2.11.1

Hotfix for UI issues

### Fixed

- No more JS errors when tabs aren't present
- All tables that have sortable attributes will get sort buttons

## v2.11.0

2.11 includes a major rewrite to the jobs engine, with a few other updates
sprinkled in.

### Fixed

- The uploads list in the front-end part of the application is now an HTML
  table, making it a lot easier to read, and fixing the odd flow problems for
  titles with long names
- Varous job-related problems have been addressed by the rewrite; see the
  "Changed" section below.

### Added

- The uploads list now shows a count of issues as well as an explanation of
  what the quick error scan actually means
- There's a new command to remove issues from disk that are associated with
  old `live_done` batches (batches which have been archived 4+ weeks ago) to
  avoid the risks of trying to identify and manually remove unneeded issues.
- There's a new, terrible, janky script: `test/report.sh`.  This script allows
  rudimentary testing of the database and filesystem states in order to act
  something like end-to-end testing for major refactors.

### Changed

- Background jobs are split up into more, but smaller, pieces.  When (not if)
  something goes wrong, it should be a lot easier to debug and fix it.
- Due to the jobs now being idempotent, all will attempt to auto-retry on
  failure.  This should mean no more having to monitor for temporary failures
  like an openjpeg upgrade going wrong, the database being restarted, NFS
  mounts failing, etc.
- `make bin/*` runs are a bit faster now, and `make` is much faster

### Migration

- All jobs have to be finished before you can upgrade from previous versions,
  because many major changes happened to the jobs subsystem.  This will require
  a few manual steps:
  - Turn off the server and worker processes.  This ensures that nobody is
    changing data, and no uploads will be processed.
  - Check the list of job types in the database:
    `SELECT DISTINCT job_type FROM jobs WHERE status NOT IN ('success', 'failed_done');`
  - Run workers for each outstanding job type, e.g., `./bin/run-jobs -c ./settings watch build_mets`
  - Repeat until no more outstanding jobs are in the database
- Run migrations prior to starting the services again

## v2.10.0

The big change here is that we don't force all titles to claim they're in
English when we generate ALTO.  Yay!

### Fixed

- JP2 output should be readable by other applications (file mode was 0600
  previously, making things like RAIS unable to even read the JP2s without a
  manual `chmod`)
- The check for a PDF's embedded image DPI is now much more reliable, and has
  unit testing

### Added

- Multi-language support added:
  - We read the 3-letter language code from the MARC XML and use that in the
    Alto output, rather than hard-coding ALTO to claim all blocks are English
  - Please keep in mind: this doesn't deal with the situation where a title can
    be in multiple languages - we use the *last* language we find in the MARC
    XML, because our internal process still has no way to try and differentiate
    languages on a per-text-block basis.
