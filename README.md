Newspaper Curation App
===

This toolsuite handles (most of) the Oregon Digital Newspaper Project's
workflow of processing born-digital PDFs and in-house scans, and then
generating batches which can be ingested into [ONI](https://github.com/open-oni/open-oni)
and [chronam](https://github.com/LibraryOfCongress/chronam).  See our other
repositories for the legacy suite.  Actually, don't, unless you want a history
lesson.  They're pretty awful:

- [Back-end python tools](https://github.com/uoregon-libraries/pdf-to-chronam)
  - This has been completely deprecated.  YAY!
- [Front-end PHP app](https://github.com/uoregon-libraries/pdf-to-chronam-admin)
  - This still has a few necessary pieces of the project.  BOO!

*Apologies*: this toolsuite was built to meet our needs.  It's very likely some
of our assumptions won't work for everybody, and it's certain that there are
pieces of the suite which need better documentation.

Preliminary Setup
---

You'll need:

- Go and some dependencies (see below)
- Poppler Utils for PDF processing
- OpenJPEG 2 for JP2 generation
- MariaDB
- A IIIF server capable of handling tiled JP2 files
- Apache for authentication as well as proxying to NCA and the IIIF server

**Please note**: The easiest way to get up and running with NCA is via
our [Docker setup](docker/).  It's fairly simple to set it up manually, but if
you go that route, it's going to be a lot easier if you read the docker files
in order to understand the installation and the stack.

Compilation requires [Go](https://golang.org/dl/) 1.9 or later, golint, and
[gb](https://getgb.io/).  Migrating the database can be done manually by
executing the "up" sections of the various migration files, but it's *far*
easier to just use [goose](https://bitbucket.org/liamstask/goose).

server
---

This is the web server which exposes all of NCA's workflow UI.  Please
note that, at the moment, this requires Apache sitting in front of the server
for authentication.

### Usage

See the [Docker setup](docker/) to understand how to install all the
dependencies, compile binaries, set up Apache, etc.  See `dev-server.sh`, or
`rhel7/nca-*.service` for examples of running and deploying NCA.

**NOTE**: The server builds a cache of issues and regularly rescans the
filesystem and the live site to keep its cache nearly real-time for almost
instant lookups of issue data.  However, building this cache requires the live
site to use the same JSON endpoints chronam uses.

ONI's JSON endpoints were rewritten to use IIIF, so out of the box, ONI isn't
compatible with this cache-building system.  The IIIF endpoints supply very
generic information, which didn't give us issue-level information without
performing thousands of additional HTTP requests, so we had to put the old JSON
responses back into our app.  If you wish to use this application with an ONI
install, you'll need to do something similar.

The relevant commit links follow:

- [Override IIIF JSON endpoints with previous JSON](https://github.com/uoregon-libraries/oregon-oni/commit/067ab17084d9015996932d2e001226aa18bbcdb6)
- [ Fix batch JSON pagination](https://github.com/uoregon-libraries/oregon-oni/commit/0463435615b23058ca1bc2afd8017e7001dc0657)
- [Fix missing route name](https://github.com/uoregon-libraries/oregon-oni/commit/94f84a30abd6ad5a38c8bd932a95297e1a9b1989)

### Administrator First-Time Setup

Your first session should be started by running the server with the `--debug`
flag, then you can hit `/sftp?debuguser=sysadmin` to log in as a super admin.
You can then set up other users as necessary.  Once you have Apache set up to
do the authentication, you shouldn't need to enable `--debug` again.

### IIIF Image Server

You must stand up an IIIF image server for metadata entry and review, as those
require visibility into the newspaper images.
[RAIS](https://github.com/uoregon-libraries/rais-image-server) is the
recommended image server due to its simplicity.

A simple invocation can be done by using the NCA settings file, since
it is compatible with bash, and has all the settings RAIS needs:

    source /path/to/nca/settings
    /path/to/rais-server --address ":12416" \
        --tile-path $WORKFLOW_PATH \
        --iiif-url "$IIIF_BASE_URL" \
        --log-level INFO

Job Runner
---

Queued jobs (such as SFTP issues manually reviewed and queued) will not be
processed until the job runner is executed.  You will want to ensure at least
one process is watching each type of job, and one process is watching the page
review folder for issues ready to be queued up for derivatives.

A simple approach to run everything needed is as follows:

    ./bin/run-jobs -c ./settings watchall

You can also run the various watchers in their own processes if you need more
granularity, but that's left as an exercise for the reader to avoid
documentation that no longer matches reality....

Other Tools
---

You'll find a lot of other tools in `bin` after compiling NCA.  Most
have some kind of useful help, so feel free to give them a try, but they won't
be documented in depth.  Most are one-offs to help diagnose problems or test
features, and shouldn't be necessary for regular use of this software.
