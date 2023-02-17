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

- Go and some dependencies (see below)
- Poppler Utils for PDF processing
- OpenJPEG 2 + command-line tools for JP2 generation
  - The command-line tools will probably need to be **manually compiled** to
    support converting PNG files.  Most distributions of Linux don't have this
    by default, hence the need to manually compile.
- MariaDB
- An IIIF server capable of handling tiled JP2 files without a ton of overhead (e.g.,
  [RAIS](https://github.com/uoregon-libraries/rais-image-server))
- Apache/nginx for authentication as well as proxying to NCA and the IIIF server

**Please note**: The easiest way to get up and running with NCA is via
our Docker configuration / setup.

- <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/docker-compose.yml>
- <https://github.com/uoregon-libraries/newspaper-curation-app/tree/main/docker>

It's not difficult to run NCA on a VM or bare metal, but if you go that
route, you'll find the docker setup helpful just in terms of understanding the
full stack and configuration.

## Compile

Compilation requires:
- [Go](https://golang.org/dl/) 1.18 or later
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

Once you've compiled, the two key binaries are going to be `bin/server` for the
HTTP listener, and `bin/run-jobs`, the job queue processor.

Note that even if you do use Docker, you'll probably want to have your dev
system set up to compile the binaries.  With a suitable
`docker-compose.override.yml` file (like the provided
`docker-compose.override.yml-example`), the binaries are mounted into the
container, allowing for quicker code changes.

## Database Setup

Creating / migrating the database can be done manually by
executing the "up" sections of the various migration files, but it's *far*
easier to just use [goose](https://github.com/pressly/goose)

    go install github.com/pressly/goose/v3/cmd/goose@latest
    goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up

*(Obviously you must replace `<user>`, `<password>`, `<db host>`, and `<database
name>` with appropriate values for your environment!)*

If you use docker, the entrypoint script should migrate automatically whenever
the container starts up.  If you're doing development and break the automatic
migration, just use the steps above inside the web container.
