P2C-go
===

This is a proof of concept toolsuite built initially to provide better insight
into the SFTP upload problems we were getting.  It's all Go because I know
it'll need a lot of refactoring (it is just another band-aid, really), and the
PHP app has been very difficult to refactor safely.

Compilation requires [Go](https://golang.org/dl/) and gb (`go get github.com/constabulary/gb/...`)

server
---

This tool currently adds two areas to the site: an SFTP reportin tool, and an
issue finder.  Both tools are information-only, and don't yet have any useful
way to actually contact publishers, reject bad issues, etc.

I am planning to fix the SFTP part up to add queuing from the web to move SFTP
issues somewhere else where a regular job would run the derivative process and
move the issues forward in the workflow.  I'm also planning something to reject
issues in some way, but it hasn't been determined the best way to handle that.

### Usage

See `dev-server.sh` or `rhel7/p2cgo.service`

Cache builder
---

For various other tools to work, a cache of all known issues must be built:

    ./bin/make-cache -c ./settings.py \
        --siteroot https://oregonnews.uoregon.edu
        --cache-path ./tmp/

Searching is fairly comprehensive.  This tool will search the live site and all
configured workflow directories to cache a list of all issues.

Issue Finder
---

This tool is a command-line-only tool for now.  The usage should suffice for
explaining how it works, but an invokation might look like:

    ./bin/find-issues -c ./settings.py \
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

Pulls a live list of all titles and prints their LCCNs and names:

    ./bin/print-live-lccns --siteroot https://oregonnews.uoregon.edu --cache-path ./tmp/
