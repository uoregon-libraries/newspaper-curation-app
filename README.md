Batch Maker
===

This is the third (and hopefully final) toolsuite for generating batches which
can be ingested into [ONI](https://github.com/open-oni/open-oni) and
[chronam](https://github.com/LibraryOfCongress/chronam).  See our other
repositories for the complete suite:

- [Back-end python tools](https://github.com/uoregon-libraries/pdf-to-chronam)
- [Front-end PHP app](https://github.com/uoregon-libraries/pdf-to-chronam-admin)

This project was created initially just to have a quick way to scan publisher's
PDFs and find errors before running the Python scripts in the other repository,
as PHP was proving unsuitable for disk scanning jobs, and Python wasn't a great
fit for a new front-end app (and neither had great error detection and
handling).  It is now planned to slowly replace the other codebases entirely,
to simplify the application as well as provide something significantly faster.

Compilation requires [Go](https://golang.org/dl/) 1.9 or later and gb:

    go get github.com/constabulary/gb/...

server
---

This tool currently adds two areas to the site: an SFTP queueing tool, and an
issue finder.  Please note that, at the moment, this requires Apache sitting in
front of the server for authentication.

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

Cache builder
---

For various standalone tools to work, a cache of all known issues must be built:

    ./bin/make-cache -c ./settings \
        --siteroot https://oregonnews.uoregon.edu
        --cache-path ./tmp/

Searching is fairly comprehensive.  This tool will search the live site and all
configured workflow directories to cache a list of all issues.

**NOTE**: As mentioned in the "server" section above, the cache won't work with
a core ONI setup, and requires putting the chronam-compatible JSON endpoints
into your application.

Issue Finder
---

This is the command-line version of the issue finder in `server`, but reports
slightly more information which should be suitable for developers / debugging.
The usage should suffice for explaining how it works, but an invokation might
look like:

    ./bin/find-issues -c ./settings \
        --cache-file ./tmp/finder.cache \
        --issue-key=sn12345678/189601

That would search for any edition of a paper published anytime in January, 1896
for LCCN "sn12345678".

At the moment, logging is overly verbose and not well-separated.  There may be
a lot of grepping needed to get useful information.

The "issue key" may consist of just an LCCN or be as complete as LCCN + year +
month + day + edition.

Error Report
---

This tool finds errors (or at least likely errors) with issues in the cache:

    ./bin/report-errors --cache-file ./tmp/finder.cache

This can be a useful way to find dangling issues that need to be remove or
fixed in some way.

Dupe Finder
---

Finds dupes for easier cleanup.  Output is a yaml list of all issue keys that
had duplicates somewhere, followed by what we believe to be the correct
canonical version and all locations seen.

    ./bin/find-dupes --cache-file ./tmp/finder.cache

LCCN list
---

Pulls a live list of all titles and prints their LCCNs and names.  It doesn't
use the cache file at this time, as it was supposed to be a throw-away
"script", but it will make use of the same JSON files the `make-cache` tool
downloads when run.  Passing in the same cache path used for that tool will
result in a significantly faster run.

Example usage:

    ./bin/print-live-lccns --siteroot https://oregonnews.uoregon.edu --cache-path ./tmp/
