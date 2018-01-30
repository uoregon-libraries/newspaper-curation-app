Batch Maker
===

This is the replacement toolsuite for generating batches which can be ingested
into [ONI](https://github.com/open-oni/open-oni) and
[chronam](https://github.com/LibraryOfCongress/chronam).  See our other
repositories for the legacy suite:

- [Back-end python tools](https://github.com/uoregon-libraries/pdf-to-chronam)
- [Front-end PHP app](https://github.com/uoregon-libraries/pdf-to-chronam-admin)

Compilation requires [Go](https://golang.org/dl/) 1.9 or later and gb:

    go get github.com/constabulary/gb/...

server
---

This is the web server which exposes all the batch-maker workflow UI.  Please
note that, at the moment, this requires Apache sitting in front of the server
for authentication.

We've provided an [example apache config](apache.conf) file which roughly
matches our own setup.

### Usage

See `dev-server.sh` or `rhel7/p2cgo.service`

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
