---
title: Not Compiling Locally
weight: 20
description: Why it's best to compile NCA locally
---

If you want to compile NCA inside a container, you are for the most part on
your own.  It's doable and pretty easy, but it's not part of the steps we're
going to outline, because it adds some annoyances.

Why is it best to compile on your local machine instead of inside a container?

- Go is not Ruby / Python / PHP / node.  You aren't installing a systemwide
  runtime or futzing with things like rbenv, nvm, virtualenv, composer, ....
- Go doesn't even really require an install per se; you can choose to download
  the binary distribution, set up some environment variables, and use it.  No
  sudo, no /usr/bin polluting, no complex compiling from sources.
- Same with go's dependencies - `go install ...` will install files in a space
  that's local to your user path.  All NCA's code dependencies are similarly
  local.  Everything "just works".  The only reason we use Docker for
  development is the various external dependencies like poppler utils, graphics
  magick, etc.
- Vim (and other editors / IDEs) usually require Go tools to be installed
  locally for code analysis, autocomplete, etc.
- This repository doesn't have a deploy system for use inside containers; if
  you want to compile inside the containers, it can be a little tricky:
  - Mount your code into `/usr/local/src/nca`
  - Within the container, go to the `/usr/local/src/nca` directory for all commands like make, gofmt, etc.
  - Copy binary files into `/usr/local/nca`

For development, it's just a lot easier to install Go locally, compile locally,
and mount the binaries inside the Docker containers.

That said, PRs which make it easier to set up a "build box" type of container
certainly would be appreciated.  Better yet, it would be lovely to get a tool
like `gin` integrated - except not a dead project, and something more
customizable.  We'd want to validate code (possibly just to report as opposed
to refusing to start the services), recompile only when `*.go` or `*.html`
files change, etc.
