---
title: Installation
weight: 20
description: How to build and compile NCA
---

## Development

If you're developing on NCA, installation will differ from standing up a
production server.  Please see our [Development Guide](/contributing/dev-guide).

## Preliminary Setup

Manual installation has several prerequisites:

- Poppler Utils for PDF processing
- OpenJPEG 2 + command-line tools for JP2 generation
  - The command-line tools will probably need to be **manually compiled** to
    support converting PNG files.  Most distributions of Linux don't have this
    by default, hence the need to manually compile.
- A recent version of GhostScript - 10+ is recommended
- GraphicsMagick
- MariaDB
- A IIIF server capable of handling tiled JP2 files without a ton of overhead (e.g.,
  [RAIS](https://github.com/uoregon-libraries/rais-image-server))
- Apache/nginx for authentication as well as proxying to NCA and the IIIF server

**Please note**: The easiest way to get a quick demo / test setup of NCA is via
our Docker configuration / setup:

- <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/docker-compose.yml>
- <https://github.com/uoregon-libraries/newspaper-curation-app/tree/main/docker>

This is great to try things out, or for a quick one-off use of NCA, but **it is
not recommended for production use**.

We strongly recommend either crafting your own containerized setup with a mind
to production reliability (which we haven't done) or just running NCA on bare
metal - one the prerequisites above are installed, the rest of NCA is very easy
to get running. If you go this route, you'll still find the docker setup
helpful just in terms of understanding the full stack and configuration.

## Compile

Compilation requires:
- [Go](https://golang.org/dl/) 1.18 or later. Go is only required for
  compilation: its runtime does not need to be installed in production as long
  as you compile on the same architecture your production system has (or change
  the `Makefile` to cross-compile for the targeted architecture).
- [revive](https://github.com/mgechev/revive): `go install github.com/mgechev/revive@latest`

The easiest way to compile is simply running `make` in the source directory.
This will grab various Go packages the application requires, validate the
current code (via revive, gofmt, and go vet, for development purposes), and
build all the binaries.

A full compilation from a clean repository should take about 15 seconds, though
this can depend on network speed the first time dependencies are pulled from
github.  Subsequent compiles generally take under 5 seconds.  If that's still
too long, and you don't mind skipping the code validations, `make fast` will
skip the validator entirely, usually saving 1-2 seconds.

If you're in a *serious* rush (say you want to auto-build every time code
changes just to see if compilation failed), you can also just build a single
binary via `make <binary target>`, e.g., `make bin/server`. This skips all
validations and only builds the binary you request, and generally takes less
than a second.

Once you've compiled, the two key binaries are going to be `bin/server` for the
HTTP listener, and `bin/run-jobs`, the job queue processor.

Note that even if you do use Docker, for development you'll probably want to
run all NCA's binaries locally and just have them communicate with the
dockerized services (IIIF server, database, and SFTPGo). Again, see our
[Development Guide](/contributing/dev-guide) for details.

## Database Setup

Creating / migrating the database is easily done using the "migrate-database"
binary compiled in a standard `make` run. This binary is a wrapper around Goose
functionality, but you will generally only need to use the "up" command.
Advanced users / devs may want to read more about [goose][goose] to learn how
it works and what other commands may be used.

[goose]: <https://github.com/pressly/goose>

```bash
make
./bin/migrate-database -c ./settings up
```

If you use docker, the entrypoint script should migrate automatically whenever
the container starts up.  If you're doing development and break the automatic
migration, just run `migrate-database` inside the web container.
