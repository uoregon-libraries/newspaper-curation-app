P2C-go
===

This is a proof of concept built initially to provide better insight into the
SFTP upload problems we were getting.  It's all Go because I know it'll need a
lot of refactoring (it is just another band-aid, really), and the PHP app has
been very difficult to refactor safely.

Usage
---

Compilation requires [Go](https://golang.org/dl/) and gb (`go get github.com/constabulary/gb/...`)

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
