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

## v6.3.0

We mostly included bug fixes in this update. Mostly.

### Fixed

- Devs: the `manage` script uses docker's config more effectively in order to
  ensure that backup and restore functions work, and process only the relevant
  docker volumes, no matter what project name you configure in your compose
  override.
- Devs: all linting / validation tooling is now properly checking all
  production code (had missed `internal` when moving packages there).
- Logging is now better classified: if you see a "CRIT" in your logs, something
  truly awful has happened, and requires manual intervention (sometimes data
  cleanup, sometimes a code bug that needs a fix ASAP).
- Fixed rare situations where a pipeline keeps doing its work when there's a
  job earlier in the sequence that hasn't finished.

### Added

- Job runner resiliency! Jobs auto-retry on failure, but only if the database
  can be contacted, because the database is where NCA stores job data. Now, in
  addition to the DB-backed retry, extremely critical paths will actually wait
  and retry when the database is unreachable. It won't wait forever, but for
  handling minor outages or a DB server reboot, this should fix a whole slew of
  very annoying data-cleanup problems.

### Changed

- Devs: minor code refactors:
  - Terminology: batches now have an activity log, not "actions". Actions are
    what a user can do, not what's been done. This only changes the field name,
    sadly, not the database misnaming here.
  - The batch view template now keeps the various states separate so it's
    (hopefully) less messy to edit the HTML for a specific batch state.
  - A handful of comment improvements
- Devs: `src/models/issue.go` had its "finder" combined and centralized with
  the one used by audit logs. Creating a new "light DSL" should be
  significantly easier now.
- Devs: added `govulncheck`, the standard Go vulnerability checking tool.
  Activate via `make audit` or manually run it.
  - e.g., `go tool vulncheck ./src/... ./internal/...`
- Devs: all tooling is now part of the `go.mod` "tool" section. This means
  these tools are auto-installed when you simply pull dependencies, and are
  invoked via `go tool <name>`.
- The top navigation links are now put into dropdown menus to allow for more
  space. The new setup is *not* something I love, because of the added time it
  takes to get to something frequently used, but we really are running out of
  space to just jam more stuff into a single "strip" at the top of the page.
- The issuefinder UI is now a bit easier to use, as it provides a typeahead /
  autocomplete search rather than a giant dropdown of all the titles.
- Minor improvements to the issuefinder "help" wording.
- All users can now use the issue finder since it's pretty safe, and has no
  destructive actions available.
- The audit log disclosure's state is now reflected in the URL, so copying and
  pasting URLs maintains not only the filters, but also whether the filter box
  is open or closed

### Removed

- Removed the following binaries: find-dupes, find-issues, make-cache, and
  rewrite-mets. These haven't been necessary for years, and/or their features
  are now in NCA.

## v6.2.0

The "roles and tools" update.

Oh, and some important test stuff got added.

This is the "roles, tools, and tests" update. Hmm.

### Added

- New role, "Site Manager", to represent a user who isn't a dev/ops person, but
  has most access in NCA.
- The role "admin" is replaced with "sysop", as in "system operator"
  (old-school term). This should be both weird enough and also clear enough to
  state this is somebody who has absolute power, not to be confused with a
  non-technical admin like the project manager.
  - As part of this, a lot of application text has been clarified or updated,
    as has our documentation.
- The database state created by NCA now includes a user with the login "sysop"
  rather than "admin", and the previous "admin" and "sysadmin" users, if they
  are in your system, will be deactivated.
  - Dev: this also changes the seed data! You'll have to adjust your URL
    parameter from `debuguser=admin` to `debuguser=sysop`.
- Dev: privileges and roles have been massively refactored to make future
  changes a lot less cumbersome.
- Dev: unit testing for privileges and roles!
- New acceptance tests have been added under `test/acceptance` as Go unit
  tests. These are extremely basic for now just to get something started, but
  they need a lot of work.

### Changed

- External tools' paths are now configurable. This will enable more people to
  use NCA than before, because custom install paths for some tools could
  prevent NCA from working. For example, it was assuming that `gm` would always
  run the Graphics Magick tools; in a handful of situations this would fail.

### Migration

- Make sure you look at the changes in `settings-example` compared to your
  `settings` file. In particular, you'll need to make sure `GRAPHICS_MAGICK`,
  `PDF_SEPARATE`, and `PDF_TO_TEXT` are set. The defaults will work for most
  people. And if you're already using NCA, the defaults will give you the same
  processing that was already happening.
- Migrate the database:
  - **Important**: if you're using "sysadmin" or "admin" today, (a) *stop doing
    this and create real users*, but (b) if you really need generic
    system-built users, **those two will be deactivated** after migrating. You
    will need to change your processes to rely instead on the "sysop" user.
  - `make && ./bin/migrate-database -c ./settings up`
- Consider downgrading people who have "sysop" to the new "Site Manager" role.
  This role has access to most things, but won't be able to perform some
  actions that are dangerous. Moving forward, sysops will get more and more
  dangerous privileges, while the site admins will only get privileges that
  are safe for a non-dev user.

### Removed

- Admins (now SysOps) no longer have access to "hack" a batch URL to view
  deleted or in-process batches. This was undocumented, so shouldn't affect
  anybody greatly, and probably shouldn't have been added in the first place.

## v6.1.1

Hotfix: "entwined jobs" (those which are group-retried on failure, like ONI
Agent jobs) use consistent restart/retry logic.

### Fixed

- Manually restarting a fatally-failed job (with `run-jobs requeue`) now
  properly restarts all entwined jobs, just like the automatic temporary
  failure retry loop does.

## v6.1.0

Better ONI Agent integration and some dev improvements

### Fixed

- Jobs involving external services (ONI Agent) are now set up such that the
  whole group is retried when something goes wrong. This fixes the "wait for
  ONI" loop when an ONI Agent job fails:
  - NCA queues a job that will call out to ONI Agent (e.g., load a batch). This
    job almost always succeeds because it's just asking the agent to put
    something in *its* job queue.
  - NCA queues a job to check ONI Agent for success. This fails if the Agent's
    job failed.
  - On failure of any job, NCA retries that job. In this case, it *only*
    retries the "check for success" job. Which just rechecks a failed Agent
    job.
  - The "check for success" job fails, and a retry is queued. But no matter how
    many times you ask "did you succeed?", if you don't start a new external
    job, the answer is still "no".
- ONI Agent's magic "job is redundant and not queued" response is now handled
  properly as a success. (e.g., queueing a batch to be loaded when it's already
  been loaded successfully)
- All MariaDB triggers use UTC time instead of local time
- All times displayed to users are local instead of some being UTC
- Devs: database initialization is a bit better in the docker setup: the
  no-longer-necessary seed file is gone and a new SQL initialization script
  exists to pre-build the DB structure that the current DB migrations create.
  This lets you skip the re-running all the migrations every time you reset the
  database (generally a pain when doing lots of testing).

### Added

- Devs: New test recipe script for helping guide a fake Solr outage test
- Devs: The integration tests now log the output of `bin/queue-batches` for
  easier debugging.

### Changed

- Devs: the `init.sql` regenerating helper function in `scripts/localdev.sh` is
  now a *destructive* operation. This is necessary to ensure a valid starting
  state for `init.sql`, rather than the situations where it ends up being easy
  to accidentally add test data.

### Migration

- Run database migrations:
  - `make && ./bin/migrate-database -c ./settings up`

## v6.0.2

Hotfix for crash when creating a title that doesn't have MARC in your first
configured MARC site (`MARC_LOCATION_1`).

- Failure to download a valid MARC record no longer **crashes** NCA. I'm not
  entirely sure how this happened, though, as Go's `net/http` is built to
  auto-recover from basically any crash.

## v6.0.1

Batch names are now stored permanently instead of calculated on the fly, and
the local development and testing is less likely to give "weird" Solr problems.

### Fixed

- Batch names are now stored in the database when a batch is built, rather than
  computed as needed. This fixes weird datestamp inconsistency when a batch is
  generated on a server with one timezone, and links to staging/production are
  generated in a different timezone. This also prevents us from having massive
  problems if we make a major change to the batch name generation algorithm.
- Devs: `scripts/localdev.sh` no longer loads titles into ONI instances
  immediately after database setup is done. This was sometimes happening faster
  than ONI could initialize the Solr schema, which caused wonderful silent Solr
  problems that would prevent loading batches, but with error messages that
  didn't really explain the root problem well.

### Migration

- Run database migrations:
  - `make && ./bin/migrate-database -c ./settings up`

## v6.0.0

Batch automation! Awardee automation!! MARC XML uploading and automation!!!

This release will *require* a new service that you run on your ONI servers
(both staging and production): [ONI Agent][oni-agent]. We're taking away the
instructions and workflow "pauses" for manually loading and purging batches, as
they will no longer be necessary. It also ensures ONI systems get any awardees
you've put into NCA, and allows you to upload a title's MARC XML into NCA to
both pre-populate a title there, and send the MARC XML to your ONI instances so
you don't have to manually load those.

In most cases, installing and configuring ONI Agents should be an easy
trade-off for the batch/title/awardee automation, but it's certainly a big
change, and will require some work to get it all up and running.

There are also a handful of dev- and test-centric improvements primarily aimed
at making it easier to build a full end-to-end test (NCA, ONI, and ONI Agent)
using docker compose.

### Fixed

- The `delete-live-done-issues` command will now report errors if a batch
  cannot be finalized for some reason
- Devs: The `localdev.sh` command for migrating the database now ensures the
  migrate command is built first
- Devs: internal refactors which affect the "public" APIs. But don't use these.
  I just haven't gotten around to moving everything to a proper "internal"
  subdirectory.

### Added

- Batch-related automation via the above-mentioned ONI Agent:
  - When a batch is ready for staging, it is automatically loaded there.
  - When a batch is ready for production, same deal.
  - When a batch needs to be purged, ONI Agent takes care of it.
  - Before a batch is sent to ONI, its awardee is checked for existence, and
    the ONI Agent attempts to create it if needed.
- NCA's "Titles" view now has a link to upload MARC XML. If you do an upload,
  two things happen:
  - NCA will create or update any record with a matching LCCN with the basic
    MARC data it cares about: LCCN, name, location, and some internal fields
    like the fact that it considers the title's LCCN to be validated.
  - The ONI agents (staging and production) will be given a command to load the
    MARC record into ONI. This will mean *no more batch failures* due to a
    missing title.
- Users with some kind of batch-related role can now see most batches, even if
  they can't take any actions.
- NCA auto-creates various workflow directories if they don't exist. This
  simplifies our docker setup as well as reducing out-of-sync situations when
  the docker entrypoint didn't properly reflect all necessary directories.
- Devs: New script for testing, `test/create-test-users.go`, which deletes
  existing users and then creates a new user for each role in NCA
- Devs: `make` now builds an ONI Agent test binary which can be used for
  various actions against an ONI instance.

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
- Major documentation overhaul, including information on the above-mentioned
  ONI Agent and automations, but also typos were fixed, obsolete data removed,
  incorred information fixed, etc.
- The "Batch Management" view groups batches in tabs so the ones that need the
  most immediate attention are easier to find.
- Most areas of NCA that used the term "purge", when referring to something
  other than the purging of a batch from ONI, have changed to some other term.
  This requires a migration (see notes below) to keep the database meaningful.
- Minimum wait times for uploaded issues to be queued are now configurable
  instead of hard-coded! Yay!
- Devs: major changes to the "automated manual" test setup to improve
  consistency (finding and renaming all batch names to something predictable)
  and ease future updates
  - You no longer need to supply your own `nca-seed-data.sql`.
  - You can put MARC XML in a `test/sources/marc-xml` directory to get them
    autoloaded when running various integration tests.
  - When running `prep_for_testing`, migration-installed users will be destroyed
    and replaced with role-based users. If you tend to test as "admin", nothing
    will appear to change, but the "sysadmin" user will no longer be available.
  - Everything related to docker compose should now reflect the best practices /
    conventions: `compose.yml` instead of `docker-compose.yml`, no "version"
    definition, etc.

[oni-agent]: <https://github.com/open-oni/oni-agent>

### Removed

- All manual flagging of batch loading and purging has been removed, as have
  the instructions for these pieces.

### Migration

#### Normal Users

- Get all batches out of NCA's "manual" workflow states prior to upgrading.
  Behavior will be undefined, but almost certainly unpleasant, if there are
  batches in some of the states that no longer exist, such as batches waiting
  for a staging purge.
  - Safe states: `live`, `live_done`, `live_archived`, `qc_ready`,
    `qc_flagging`, or `deleted`
  - Unsafe: `pending`, `staging_ready`, or `passed_qc`
- Configure NCA's new settings:
  - `STAGING_AGENT`: The *ssh endpoint* (not necessarily your public URL if you
    have something like HAProxy in front of your servers!) to your staging
    server's ONI Agent, e.g., "staging.oregonnews.uoregon.edu:2222"
  - `PRODUCTION_AGENT`: The ssh endpoint to your production server's ONI Agent
  - `DURATION_ISSUE_CONSIDERED_DANGEROUS`: How long after upload an issue must
    wait before it's allowed to be queued for curation
  - `DURATION_ISSUE_CONSIDERED_NEW`: How long after upload an issue gives a
    warning about being new
- Install the ONI Agent onto your production and staging systems
  - Consider compiling NCA via `make` somewhere off production prior to
    deploying, and trying out `bin/agent-test`. Running this command will give
    you an explanation of how to use it to test your NCA and ONI Agent setup.
    This will help catch firewall issues, configuration mishaps, etc.
- Shut down NCA workers and HTTP daemon
- Delete the NCA issue cache (e.g., `rm /var/local/news/nca/cache -rf`)
- Run database migrations:
  - `make && ./bin/migrate-database -c ./settings up`
- Restart services

#### Developers

- The docker stack has been rewritten pretty much from scratch so that it's
  more like a production setup and includes *all* dependencies now, including
  ONI and ONI Agent for NCA to send its various commands.
- You *will* need significant changes to your docker setup! There is *no*
  direct migration path.
  - If you use docker compose, you probably need to rename your override to
    `compose.override.yml`
  - There are now two example compose overrides, and they are are a lot more
    "out of the box" than previously. Copy the one that makes the most sense
    and you should only need a few tweaks.
  - Review your settings file if you're doing hybrid dev. With all the ONI
    services now in containers, various setting will need to change and some
    might need to change depending on how you're set up. It's best to do a full
    review, but below are some of the key settings:
    - `STAGING_AGENT` and `PRODUCTION_AGENT` will definitely need to change to
      point to the correct agent. One of those settings may be fine, but the
      two can't be the same anymore, so one has to change.
    - `STAGING_NEWS_WEBROOT` should change so you can easily verify NCA is
      loading batches into the right environment when they're QC-ready.
    - `BIND_ADDRESS` may need to change if it conflicts with any of the new
      services (or else you'll need to change those services in the compose
      override file)
    - `SFTP_API_URL` will probably have to change, or you'll need to change the
      compose override file, as its port is 8081 in the example overrides.
  - You should probably remove all existing docker images, volumes, containers,
    etc. You should delete your local issue cache if you mounted one.
  - A lot of environment variables are gone now, such as `APP_URL`. If you used
    `.env` you probably need to translate it based on the new setup, using the
    override of your choice as a starting point. There's a good chance you
    won't need to do anything, though, as most docker setup is a lot easier now
    that there are two examples to choose from.
  - If you had a custom nca-seed-data.sql, you should rename it, and you'll
    have to abandon it or manually load it when needed. This part of the
    integration suite is no longer done for you to make it easier for a typical
    developer to just dive into the tests.
  - Re-read basically everything related to testing! A lot has changed.

## v5.0.1

Batching is now *allowed*! WOW!

### Fixed

- Users can now be properly assigned the "Batch Builder" role. Previously, only
  admins could generate batches.

## v5.0.0

Batching and Boostrap 5.

This release includes a major UI overhaul, getting us up to Bootstrap 5, as
well as a lot of improvements to how batching is done! You can now generate
batches from the UI!

And of course quite a few fixes.

### Fixed

- Curated issues no longer show up briefly on the curator's desk with an
  "awaiting processing" message
- Double-submitting of forms is now properly prevented
- When SFTPGo is unavailable, certain pages will give errors still, but NCA
  will at least no longer fail to start

### Added

- New area in NCA's web UI for generating batches! The command-line tool will
  continue to exist, but this UI makes it so that people without shell access
  can easily generate batches on demand.
- Various views now include issue and batch page counts
- New script in `scripts/export-db.sh` for doing a partial database export,
  suitable for exporting live data for dev use. The export only grabs the last
  month's worth of data for the largest tables, so your exports will be usable
  for testing real-world data, and still very large, but not infeasibly so.

### Changed

- Updated to Bootstrap 5 CSS
- Dark mode is now respected if the OS has it set
- Sortable table headers now look consistent with other buttons
- The nav item "Batches" has been renamed to "Manage Batches" so it's purpose
  is clearer now that we have a second batch-related nav item.
- Batch generation had a minor refactor as part of the preliminary work for
  batch generation from the website. The end results are still the same, and
  changes should go mostly unnoticed, but there are a few differences to how
  issue queues get split into batches, the command-line tool's logging is
  different, etc.
- devs: in debug mode, some HTML templates will automatically reload
- devs: Bootstrap CSS is loaded in a layer so that custom rules always take
  priority. No more `!important` nonsense!

### Removed

- Bootstrap JS is no longer used
- jQuery (and a very old version at that) is no longer used

### Migration

- Shut down NCA workers and HTTP daemon. You will have potentially several
  minutes of downtime.
- Run database migrations. This will add the new page count field, and then
  count pages for every issue in the database, which can take a while if you
  have a lot of pages:
  - `make && ./bin/migrate-database -c ./settings up`
- Restart services.

## v4.4.0

Better batching, and a tiny bit of cleanup.

Batch generation is now much faster, as the SHA sums are generated after each
issue is curated instead of all at once on batch generation. Additionally,
batches minimum/maximum page count can be overridden on the command line to
allow one-off batching runs without modifying settings.

### Fixed

- tests: reports run on separate days should be a lot easier to compare
- tests: passed-in test names work again

### Added

- `queue-batches` now optionally takes command-line flags to override the
  min/max batch size settings. This was done to allow cron jobs that behave
  differently than a manual run.
- jobs: added simple unit tests for bagit jobs
- The "Find Issues" view explains why a new batch's issues might not show up
  for a while

### Changed

- After curation, a `.manifest` file is generated in the issue's directory
  which contains the SHA256 sums of all files in the issue. This data is then
  used when batches are generated to significantly reduce the time it takes to
  finish generating a batch.

### Removed

- Removed a variety of dead functions and structs

## v4.3.2

Minor improvements.

### Fixed

- Uploaded issues can no longer be doubly queued for ingest into NCA. This was
  extremely rare with the "prevent double submit" JS, but it did manage to
  happen a couple times for us.
- Test recipes no longer run workers needlessly after batch approval. Instead
  they instruct you to mark the batches live so the full process can be
  completed and tested (marking the batch live and setting it ready for
  archive).

### Changed

- Upgraded the docker SFTPGo version to 2.5
- Changed how NCA connects to SFTPGo on user updates to match the way SFTPGo
  2.5 works. This should be compatible with older SFTPGo versions.

### Migration

- If you're running an older version of SFTPGo you may have to upgrade to
  v2.5.0. I expect the NCA changes to be backward-compatible, but I'm not 100%
  certain.
- If you have a long-running dev setup for some reason, your SFTPGo database
  won't work with this update. You should generally just be okay to start
  fresh, as the docker setup isn't meant for long-term data storage, but if you
  absolutely need to keep your SFTPGo data intact, you'll have to either
  override the SFTPGo version (in `docker-compose.override.yml`) or figure out
  how to do the data upgrade manually. SFTPGo only works when migrating data
  one minor version at a time. e.g., assuming you were on v2.3, you'll need to
  get SFTPGo v2.4, migrate, then go to 2.5 and migrate again.

## v4.3.1

Batch purging. That's all.

### Added

- Batches which fail QC can now be purged entirely, pushing all issues back to
  NCA so they can be rebatched. Sounds crazy? I thought so, too, until
  discovering that we had built a batch by mistake.

## v4.3.0

This one goes out to the devs I love. This one goes out to the devs I've left
behind.

Oh also there are a handful of nice fixes for non-devs.

This time I'm splitting up the changelog by dev vs. normal person since most NCA
users won't care about dev changes.

### Regular people

#### Changed

- The issue queue functionality that automatically moves scanned issues has
  been improved to not log critical errors when an issue has a pending job
  associated with it. Heavy workloads and/or slow filesystems can take hours to
  get huge scanned issues into NCA, and it's not an error to simply have a job
  that hasn't started yet.
- The batch queue command-line app will no longer allow issues that are tied to
  unvalidated titles into a batch. This means that a title *must* have a valid
  LCCN that exists elsewhere (usually Chronicling America or your production
  ONI site).
  - Batching a title that has an invalid LCCN is almost always a mistake, as ONI
    won't be able to ingest it, so this shouldn't affect many people in practice.

#### Migration

- If you use titles without MARC records somehow, you'll need to either start
  generating valid MARC records or else manually edit the database in NCA to
  flag titles as having been validated. *This is not recommended*, and therefore
  there will be no instructions for doing so.

### Devs

#### Added

- New test recipe created to specifically test that queue-batches properly
  ignores titles with unvalidated LCCNs

#### Changed

- The docker build is now based on RockyLinux 9 to match UO's production setup
  more closely. It's still recommened that you use a local development setup,
  and of course for production docker-compose isn't a great idea, but it's
  helpful to test things quickly and validate a new environment (in this case
  RockyLinux).
  - The build process is now *significantly* simpler. You can see the image
    definition yourself in `docker/Dockerfile-app`, but the gist is that we no
    longer compile a patched version of poppler, nor install openjpeg2 tools
    from source.
- The docker build forcibly overwrites the settings file's SFTPGo API key on
  *every run*. This eases dev / testing in some situations, but again makes it
  a bad idea to use docker in production.
- The docker override example file is a little smarter: "bin" is not mounted
  inside the container, as that is the cause of many a headache; and
  "/mnt/news" is not assumed to exist on the host
- Various minor fixes to things only VSCode seemed annoyed about. VSCode users
  should have a much easier time working on NCA. Party time? 🥳

## v4.2.0

It's all about the batches. And jobs in general. Oh, and docs. Oh, right, and a
few miscellaneous fixes.

### Fixed

- Issue keys are even more carefully validated when dupe-checking
- Incorrect toggling of a batch state exposed a batch for processing (ready for
  quality control) when it still had jobs in the queue. It was extremely
  unlikely somebody would get to the batch and do anything with it in between
  jobs, but it was still a possibility.
- When a user rejects a batch, the process is now a bit more streamlined in the
  codebase, making database errors less annoying if they do occur.
- Manual batch loading instructions are now correct for all cases (staging or
  production) due to the changes below.
- Documentation mega-overhaul! The theme is now 100% keyboard-accessible, and
  the documentation has been given a full audit, and rewritten to match what
  NCA actually does.

### Added

- Dev: The job runner now has a special flag that runs a single job and then
  exits. This can help identify which job is going rogue: run a job, check
  database state, run the next job, check state, etc.

### Changed

- Dev: jobs now have a way to signal more than just success or temporary
  failure. This is primarily needed for the upcoming API jobs where we need a
  "not failed, but wait and retry" status while waiting on ONI to complete a
  task, but also allows the rare fatal failure to signal the processor not to
  retry a job at all.
- Once batches are built, their "always-online" files are immediately synced to
  the live location rather than waiting for QC approval. This improves the
  typical case where a batch is approved and loaded to production, but it also
  simplifies batch processing in general. Staging and production servers can
  use a unified location for batch loading, and staging won't need a batch
  purge simply to reload it from the final path.
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

## v4.1.2

Page labels can be entered by real humans again!

### Changed

- Dev:
  - The docker image is now based on Fedora 38
  - Docker and CircleCI now use Go 1.21

### Fixed

- Page labeling is fixed in the UI. TIL I need to test using the front-end app
  more often, not just via automated SQL-based data entry.
- Dev: added `workonce` function to `scripts/localdev.sh` for running workers
  with `--exit-when-done` for a quicker one-off job run

## v4.1.1

Codename: "small fixes to stuff that's been lingering far too long."

### Fixed

- Various dev- and CI-specific improvements
- Page labeling no longer breaks when a page's label includes a comma
- When flagging issues in a rejected batch, already-flagged issues are no
  longer able to be reflagged (causing an error)
- In the issue upload screen, issues with an invalid folder name no longer
  cause a flurry of incorrect "This issue appears to be a duplicate..."
  messages.

### Migration

- Shut down NCA workers and HTTP daemon
- Run database migrations to fix all existing page labels:
  - `make && ./bin/migrate-database -c ./settings up`
- Restart services

## v4.1.0

Pipelines, filesystem job refactoring, and batch actions.

This update is primarily focused on improving the syncing of batches across
filesystem boundaries. All sync/verify jobs are completely redone to make
failures (e.g., mount points dropping, CIFS file corruption issues) less
cumbersome to fix.

A minor but useful change was also added to give an activity log (via NCA's
"actions" DB table) to batches. This should make it a bit easier to know how to
handle a batch, particularly when one goes to QC, gets rejected, and then
returns to QC. In many cases the second round of QC can be a lot quicker than
the first, but only if the QCer remembers this batch was already examined.
Action logs should help there.

Testing had some major changes as well, making it easier to do the
dev/test/report loop. Which affects nobody but me. Still, pretty great stuff!

### Fixed

- Devs: `test/report.sh` strips more DB ids correctly, making test reporting
  far easier to scan for real changes

### Added

- Database migrations are now easier to run and more self-contained in a new
  binary that reads DB settings from your configuration.
- New flag for job runner to auto-exit when there are no more jobs to run. This
  is primarily to enable fully scripted testing and shouldn't be used in
  production environments.
- The batch page now displays an action log, similar to issues' actions. These
  have been named "activity logs" now, to make it less confusing (batches have
  a section for "Actions" already, and these so-called "actions" are really
  just various activities people and NCA have done to an issue or batch).

### Changed

- Background jobs have been fundamentally changed:
  - All jobs belong to a "pipeline"
  - A pipeline is a group of jobs built to accomplish a given high-level
    process in NCA, such as preparing a PDF for page renumbering, building a
    batch out of a set of issues, etc.
  - Non-devs shouldn't notice any change in NCA!
  - Devs will be able to query the database to more easily find grouped jobs
    for debugging, seeing what's still pending for a pipeline they're waiting
    on, etc.
  - Eventually we hope this helps us create a UI where you can see a pipeline
    and all its related jobs and their statuses, runtime, etc.
- All jobs which were previously `SyncDir` have been split into two:
  - SyncRecursive is a light-weight, self-replicating job designed to take on
    the majority of what `SyncDir` used to do. For a given source and
    destination, all regular files (which don't exist or are a different file
    size) are copied without any post-copy validation. All directories are
    aggregated and queued up as new `SyncRecursive` jobs to be run as
    "siblings" (same priority) in the pipeline.
  - VerifyRecursive validates the SHA256 hash of every file in the source
    directory matches that in the destination directory, re-copying any which
    didn't. This is exatly the same as the prior `SyncDir` job, it just has
    less work to do since the new `SyncRecursive` job(s) will do the initial
    copying.
- Devs: the "general" test recipe has been rewritten as a bash script rather
  than a document, and can be run directly instead of reading its contents and
  adjusting it as needed.
  - As with all test recipes, you still need to be using the "advanced" dev
    setup for this to work.
  - The general test now does a bit more work and has an example of a way to
    "resume" a test if existing backups are already on disk, allowing one to
    build up a test gradually and/or add to an existing test without starting
    all the way from the beginning.
- Database IDs have more space (`BIGINT` instead of `INT`):
  - `job_logs` and `jobs` have been altered to allow for a much larger maximum
    ID to avoid catastrophic failures in the event one ever reaches ~2 billion
    entries. Unlikely to happen even after a decade of heavy use, but the new
    filesystem jobs will be a lot more numerous and have a lot more logging,
    making this risk slightly more concerning.
  - NCA's codebase has been updated to handle not only the `BIGINT` fields for
    jobs and job logs, but also in any future table just in case we need bigger
    ids elsewhere.

### Migration

- Drain the job queue entirely! This means no jobs should be in any status
  other than `success` or `failed_done`.
  - Check the database manually: `SELECT * FROM jobs WHERE status NOT IN
    ('success', 'failed_done');`
  - Don't add anything new to the PDF / scanned issue folders
  - Wait for pending and on-hold jobs to complete
  - Requeue any `failed` jobs or else cancel them (e.g., with
    `purge-dead-issues`)
  - **Note**: If you leave any jobs in any status other than `success` or
    `failed_done`, the database migrations will refuse to run and you won't be
    able to start up the NCA server.
- Turn off all NCA processes
- Back up your database, and make sure your backup has all the triggers in it,
  not just a simple data export!
- Run database migrations. *Note: this won't run if you don't drain the job
  queue first.*
  - `make && ./bin/migrate-database -c ./settings up`
- Start NCA services back up

## v4.0.1

Hotfix for possible issue that may or may not be in my head.

### Fixed

- This removes the iffy "quiet retry" feature when syncing directories, which
  is responsible for annoyance and possibly file loss. At best, it's definitely
  annoying. At worst, it's silencing problems we need to be aware of.

## v4.0.0

The NCA 4 release fixes and improves a lot of areas, but there are some
*exciting* and **amazing** things you'll really want to look at:

- Probably the biggest change is our SFTPGo integration and documentation! This
  means you don't have to try and figure out how to get publishers' content
  into NCA if you just stand up an SFTPGo instance and "connect" NCA to it.
- The other major improvement is the batch management pages, adding a slew of
  features to reduce manual command-line work when creating, copying, and
  archiving batches.
- Massive changes to error handling and code resilience in general. Users won't
  likely notice anything, but some rare crashes have been dealt with.
- Manual testing is a whole lot easier, and even has a good deal of
  documentation now. Users won't care, but devs will <3 this.

**Read the migration notes carefully**! There's a lot of change in here!

Exciting! Amazing!

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
- All issue- and batch-specific jobs are first setting the object's state and
  saving it, and only on success queueing up jobs. This fixes rare issues where
  a slow or dead job runner would allow a user to try to take action on an
  issue/batch that was already scheduled to have a different action taken.
  Rare, but disastrous.
- "Finalize" button in batch status page now queues up a batch deletion job if
  all issues in a batch were flagged for removal
- Directory sync jobs now retry during the job a few times before an error is
  returned. This should help with small I/O "hiccups".

### Added

- [SFTPGo](https://github.com/drakkan/sftpgo) is now integrated with NCA for
  managing titles' uploads.
- Batch Management:
  - Various instructions and status-setting buttons have been added to the
    batch management page for batch loaders
  - Instructions for batch loaders' manual tasks now have a "copy" button,
    which should make manual tasks a bit easier
  - Batches which were once on staging now have to be marked as purged from
    staging before they can move to other statuses (e.g., loading to
    production)
  - A new setting, `BATCH_PRODUCTION_PATH`, has been introduced. Set this to
    the location NCA should copy your batches when they're ready for being
    ingested into production.
    - On QC approval, batches are automatically synced to the location
      specified by the new setting (`BATCH_PRODUCTION_PATH`).
  - A new setting, `BATCH_ARCHIVE_PATH`, has been introduced. Set this to the
    location NCA should move your batches after they're live.
    - Once a batch is marked live, NCA will kick off a job to move all files
      out of NCA and into this location.
  - Batch loaders can now mark issues as live, which moves them to the
    aforementioned location, and as archived, which allows
    `delete-live-done-issues` to remove their workflow files (after a delay).
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
  - The `manage` script restarts key services after shutting them down and use
    helper scripts when manually testing NCA in a real-world-like setting
  - New test script to enter and review dummy metadata for quicker testing
  - New documentation created to help devs create new configuration settings.
  - New documentation added to the "test" directory to help explain how to
    create data, and in-depth "recipes" for manual testing. (See the `test/`
    directory's `README.md`)
  - New Makefile recipe for more easily testing hugo doc changes

### Changed

- Users with the role "Title Manager" can now edit all aspects of a title,
  including SFTP data. Since we no longer store plaintext passwords, there's no
  reason to do the weird half-editing setup we had before where only admins
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
- Derivative-generating jobs now fail after only 4 retries (5 tries total)
  instead of 25 (26 total). Failures with these jobs are almost always fatal,
  and we want them out of NCA sooner in order to fix the underlying problems
  manually (e.g., a corrupt PDF).
- For devs: the `jobs` package no longer exposes a bunch of low-level
  functionality for a more predictable app. Nothing outside `jobs` can just
  toss random jobs into a queue without creating a high-level function.
- The `purge-dead-issues` command is now very basic as a result of needing to
  stop letting it do very low-level work with the `jobs` package. The end
  results are the same, it just doesn't give as much output and no longer has a
  dry-run default.

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
