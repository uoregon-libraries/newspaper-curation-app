Black Mamba, The Batch Maker
===

This is the replacement toolsuite for handling the workflow of processing
born-digital PDFs and in-house scans, and then generating batches which can be
ingested into [ONI](https://github.com/open-oni/open-oni) and
[chronam](https://github.com/LibraryOfCongress/chronam).  See our other
repositories for the legacy suite.  Actually, don't, unless you want a history
lesson.  They're pretty awful:

- [Back-end python tools](https://github.com/uoregon-libraries/pdf-to-chronam)
- [Front-end PHP app](https://github.com/uoregon-libraries/pdf-to-chronam-admin)

Compilation requires [Go](https://golang.org/dl/) 1.9 or later and gb:

    go get github.com/constabulary/gb/...

server
---

This is the web server which exposes all of Black Mamba's workflow UI.  Please
note that, at the moment, this requires Apache sitting in front of the server
for authentication.

We've provided an [example apache config](apache.conf) file which roughly
matches our own setup.

### Usage

See `dev-server.sh` or `rhel7/blackmamba.service`

**NOTE**: The server builds a cache of issues and regularly rescans the
filesystem and the live site to keep its cache nearly real-time for almost
instant lookups of issue data.  However, building this cache requires the JSON
endpoints chronam uses.  ONI was rewritten to use IIIF instead of the old JSON,
and, out of the box, ONI isn't compatible with this cache-building system.  The
IIIF JSON supplies very generic information, which doesn't give us enough to
report very well on any given issue, so we had to put the old JSON responses
back into our app.  The relevant commit links follow:

- [Override IIIF JSON endpoints with previous JSON](https://github.com/uoregon-libraries/oregon-oni/commit/067ab17084d9015996932d2e001226aa18bbcdb6)
- [ Fix batch JSON pagination](https://github.com/uoregon-libraries/oregon-oni/commit/0463435615b23058ca1bc2afd8017e7001dc0657)
- [Fix missing route name](https://github.com/uoregon-libraries/oregon-oni/commit/94f84a30abd6ad5a38c8bd932a95297e1a9b1989)

If you wish to use this application with an ONI install, you'll need to do
something similar.

### IIIF Image Server

You must stand up an IIIF image server for metadata entry and review, as those
require visibility into the newspaper images.
[RAIS](https://github.com/uoregon-libraries/rais-image-server) is the
recommended image server due to its simplicity.

A simple invocation can be done by using the Black Mamba settings file, since
it is compatible with bash, and has all the settings RAIS needs:

    source /path/to/black-mamba/settings
    /path/to/rais-server --address ":12416" \
        --tile-path $WORKFLOW_PATH \
        --iiif-url "$IIIF_BASE_URL" \
        --log-level INFO

If using the [RAIS Docker Image](https://hub.docker.com/r/uolibraries/rais/),
the approach is still fairly straightforward, but we haven't set this up yet.
We're hoping to give Black Mamba a full docker-compose overhaul soon.

Job Runner
---

Queued jobs (such as SFTP issues manually reviewed and queued) will not be
processed until the job runner is executed.  You will want to ensure at least
one process is watching each type of job, and one process is watching the page
review folder for issues ready to be queued up for derivatives.

A simple approach to run everything needed is as follows:

    ./bin/run-jobs -c ./settings watchall

You can also run the various watchers in their own processes if you need more granularity:

    # One worker just watches the file-move jobs since these are heavy on IO but not CPU
    ./bin/run-jobs -c ./settings watch sftp_issue_move move_issue_for_derivatives

    # One worker for page-split jobs and derivative generation since they're both going to fight for CPU
    ./bin/run-jobs -c ./settings watch page_split make_derivatives

    # You MUST have *exactly one* worker watching the page-review folder
    ./bin/run-jobs -c ./settings watch-page-review

Black Mamba?
---

"Black Mamba" makes no sense for this toolsuite, so what is the rationale
behind the name?  Well, first, a [black mamba](https://en.wikipedia.org/wiki/Black_mamba)
is a really cool snake.  Really cool snakes are much more awesome than
random non-English words or phrases when it comes to naming projects primarily
used by native English speakers.  This is an indisputable, objective fact.
Second, when this project bites you, well... it hurts.  A lot.

It's also the same acronym as "Batch Maker", so that's fun.
