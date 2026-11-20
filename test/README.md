# Testing NCA

For a "realistic" test, NCA has a separate test stack and tooling that
creates and manipulates fake data, without a whole lot of repetitive data
entry.

## The Test Stack

The test stack is a second copy of the dev stack (see the [Developer's
Guide][dev-guide]) that's configured entirely by the repository: it ignores
your `compose.override.yml` and variables exported in your shell, and the only
thing it takes from your `.env` is `PUBLIC_HOST`, since that's about where
your browser is (e.g., NCA runs on a VM), not how the stack behaves. So a test
behaves the same for everybody, and running one never touches your dev stack's
data. Both stacks can run at once.

`bin/ncatest` runs it. Build it (and `bin/ncadev`, its dev-stack twin) from
the NCA root with Go on your host:

```bash
make tools
bin/ncatest help
```

`ncatest` works from anywhere in the checkout, but the docs assume you're in
the NCA root.

The test stack:

- is the compose project `nca-test`, so its containers, volumes, and images
  are separate from the dev stack's
- uses `compose.yml` with `test/stack/compose.yml` in place of your override
- is configured by `.env.example`, with the test stack's own settings (ports,
  paths, and anything the tests depend on) replacing some values. `ncatest`
  writes the result to `test/var/stack.env` each time it runs; see `testEnv`
  in `tools/ncadev/internal/devtool/stack.go`.
- keeps NCA's workflow tree in `test/fakemount`, and everything else it keeps
  on the host (logs, issue cache, save points) in `test/var`
- publishes NCA at <http://localhost:4333/nca>, production ONI at
  <http://localhost:4080>, staging ONI at <http://localhost:4082>, and SFTPGo's
  web admin at <http://localhost:4081> (with your `PUBLIC_HOST` in place of
  "localhost", if you've set one). Debug mode is on, so you can act as any
  user with a query parameter, e.g.,
  <http://localhost:4333/nca?debuguser=sysop>.

The first time NCA starts in the test stack, it builds its issue cache. That
takes ten minutes or more, unless your dev stack has an issue cache, which the
test stack copies to start its own.

[dev-guide]: <https://uoregon-libraries.github.io/newspaper-curation-app/contributing/dev-guide/>

## Source Data

To start, you need source data. We have [a repository of Oregon data][1] for
this. It's obviously very Oregon-specific, but it could be useful to look at or
as a starting point if you change some directory names to match your actual
titles / awardees.

[1]: <https://github.com/uoregon-libraries/nca-test-data>

All issues you want to bring into a test NCA instance will live under
`sources/scans` or `sources/sftp`. The `scans` dir would be for issues which
have TIFFs and PDFs, where the TIFF is the record of source and the PDF has the
OCR data. `sftp` is for born-digital issues which only have PDFs.

Naming conventions must be adhered to for the test code to properly put issues into NCA:

- Scanned issues: `sources/scans/<MARC org code>-<LCCN>-<Date><Edition>`.
- SFTPed issues: `sources/sftp/<LCCN>-<Date><Edition>`.

The MARC org code is going to be something like "oru", and is typically three
letters. LCCN is always 10 characters. Date is 8 characters in the form of
`YYYYMMDD`, so January 2nd, 2006 would be `20060102`. Edition is a two-digit
value for the issue's edition in case there were two editions published the
same day. This is almost always `01`.

Examples:

- `sources/sftp/sn83008376-2017011301`: The January 13th, 2017 edition of *The
  Daily Astorian*.
- `sources/scans/oru-sn96088073-1801050801`: The May 8th, 1801 edition of *The
  Bohemia Nugget*, using the MARC org code representing "University of Oregon
  Libraries".

---

*Note*: If you have titles that can't be found on Chronicling America, or you
simply don't want to add them to NCA yourself, **you also need to provide MARC
XML records** in the `sources/marc-xml` directory. The filenames have to end in
either `.xml` or `.mrk`.

If you don't provide these, you won't be able to queue up batches in NCA
without manually adding the titles. And if they aren't available on Chronicling
America, batches will fail to load into ONI.

Resetting the stack (`bin/ncatest reset` or `prep`, below) loads these.

### Want A Subset?

This directory's `.gitignore` is set up so that anything beginning with
"sources" will be ignored from any git commands. If you want, you could have a
huge list of source issues in something like "sources-all", and just copy in a
subset of issues when you are testing a specific situation. This can make the
test runs significantly faster.

## Ingest Sources

`bin/ncatest prep` gives you a fresh stack with the test issues loaded:

- Deletes everything in the databases (NCA's, both ONIs', and SFTPGo's)
- Seeds NCA's database with two MARC Organization codes (awardees), so the
  NCA test data repository (see above) works out of the box, and one user per
  distinct NCA role
- Loads all titles (from `test/sources/marc-xml`) into NCA, as well as both
  ONIs
- Empties the workflow tree (`test/fakemount`), then copies the source PDFs
  and TIFFs into it in the structure NCA requires
- Queues every born-digital issue for page review, the way a curator would
  (scanned issues don't need it: NCA picks them up on its own, since in-house
  scans are assumed to be built to spec)

`bin/ncatest reset` does all of that except loading test issues.

`bin/ncadev prep` does the same to your dev stack, if you want test issues to
play with there.

## Test Steps

Each step that would otherwise mean a human clicking through NCA has a
command. They run in a throwaway container (the test harness, in
`test/harness`), so they work on the stack's database and workflow tree
directly. `bin/ncadev` has the same commands for the dev stack.

### Run Workers

The `workers` service runs jobs as soon as the stack is up. To watch them run
in the foreground instead, use `bin/ncatest workonce`: it stops the service
and runs jobs until there are none left.

Depending on how many sources you defined, and what type, this could take no
time or it could take a very long time. Scanned images are processed the moment
NCA sees them, which means derivatives have to be generated. SFTPed issues,
however, are simply pre-processed briefly (pages are split and converted to
PDF/a) and then moved to the page review directory.

When all jobs are complete, you should see all *valid* issues moved either to
the NCA workflow location (scanned issues) or the page review location (sftp
issues). To check on jobs by hand: `bin/ncatest sql -e "select * from jobs
where status not in ('success', 'on_hold')"`.

### Page Review

`bin/ncatest rename-page-review` fakes a page review pass: it renames each
issue's PDFs to NCA's naming spec (0001.pdf, 0002.pdf, etc.) and writes a
manifest for it (so you don't have to wait for NCA's server to do that).

To tell NCA these issues were processed a while ago, run `bin/ncatest
make-older`. It rewrites every issue manifest to say the issue was last changed
four days ago (or however many days you give it), which allows you to queue
these issues for processing.

The job runner checks the filesystem fairly infrequently, so if it's already
running, it could take several minutes for the issues to make their way into
NCA. `bin/ncatest workonce` starts a fresh one.

### Curation and Review

If you're looking to test things that come after metadata entry and/or metadata
review, two commands stand in for curators and reviewers:

- `bin/ncatest curate` enters dummy metadata for every issue awaiting metadata
  entry, and queues it for review
- `bin/ncatest review` approves every issue awaiting metadata review, and
  queues the job which finalizes the metadata (generating METS XML)

Once those jobs run, issues will be ready for batching via the standard
`queue-batches` command (`bin/ncatest cli queue-batches`).

### Reports

`bin/ncatest report <name>` writes `test/<name>-report`: a list of every file
in the workflow tree, XML files and job logs scrubbed of dates and IDs, TIFF
checksums, and dumps of key database tables. Batch names are replaced with
names based on what the batches contain, since jobs can run in a different
order from one run to the next. Two runs' reports should be the same unless
something really changed, so you can diff them (see the recipes' README).

### Save Points

`bin/ncatest save <name>` saves the stack's state, its workflow tree and data
volumes (databases, Solr indexes, etc.), to `test/var/savepoints/<name>`, and
`bin/ncatest restore <name>` puts it back. This is handy for repeating a test
from the same starting point. Go's caches and NCA's issue cache aren't saved,
since they aren't application state.

## Recipes

The [recipes subdirectory](./recipes) has test recipes. Some are built into
`ncatest` (`bin/ncatest recipes` lists them): `bin/ncatest run <recipe>
<name>` runs one from scratch, makes a save point after each stage, pauses when
it needs you to check (or break) something by hand, and finishes with a report.
Others are step-by-step instructions to follow by hand. Either way, they
probably won't work exactly as written for other situations, but they should be
a useful guide to help construct your own tests.
