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

Issue Finder
---

This tool is a command-line-only tool for now.  The usage should suffice for
explaining how it works, but an invokation might look like:

    ./bin/finder -c ./settings.py \
        --siteroot http://oregonnews.uoregon.edu \
        --cache-path /tmp/p2c-cache \
        --issue-key=sn12345678/189601

That would search for any edition of a paper published anytime in January, 1896
for LCCN "sn12345678".

At the moment, logging is overly verbose and not well-separated.  There may be
a lot of grepping needed to get useful information.

Searching is fairly comprehensive.  This tool will search the live site and all
configured workflow directories to find an issue, and the "issue key" may
consist of just an LCCN or be as complete as LCCN + year + month + day
+ edition.
