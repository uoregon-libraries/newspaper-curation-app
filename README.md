P2C-go
===

This is a proof of concept toolsuite built initially to provide better insight
into the SFTP upload problems we were getting.  It's all Go because I know
it'll need a lot of refactoring (it is just another band-aid, really), and the
PHP app has been very difficult to refactor safely.

Compilation requires [Go](https://golang.org/dl/) and gb (`go get github.com/constabulary/gb/...`)

SFTP Reports
---

This tool reports likely problems with SFTP uploads.  It is information-only,
and doesn't yet have any useful way to actually contact publishers, reject bad
issues, etc.

I am planning to fix this up to add queuing from the web to move SFTP issues
somewhere else where a regular job would run the derivative process and move
the issues forward in the workflow.  I'm also planning something to reject
issues in some way, but it hasn't been determined the best way to handle that.

### Usage

For use in production, the following bash works.  Port 12345 is forwarded in
Apache so that /odnp-admin/sftpreport hits this app instead of the PHP app.  If
things can slowly migrate to this app, we'll eventually just forward
/odnp-admin to this app.  If we can't migrate everything, we will want to
instead pull this app's functionality into PHP.

    # I compiled the server and renamed bin/sftp-report to "server111" so it's
    # clear we're running v1.1.1
    ./server111 \
        -c /home/jechols/projects/pdf-to-chronam/src/config/settings.py \
        -p 12345 \
        --webroot=/odnp-admin/sftpreport \
        --parent-webroot=/odnp-admin \
        --static-files $(pwd)/static \
        $(pwd)/templates

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
