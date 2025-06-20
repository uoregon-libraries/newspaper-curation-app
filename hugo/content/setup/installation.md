---
title: Installation
weight: 20
description: How to build and compile NCA
---

## Development

If you're developing on NCA, installation will differ from standing up a
production server. Please see our [Development Guide][dev-guide].

[dev-guide]: <{{% ref "/contributing/dev-guide" %}}>

## Preliminary Setup

Manual installation has several prerequisites:

- Poppler Utils for PDF processing
- OpenJPEG 2 + command-line tools for JP2 generation
- GhostScript
- GraphicsMagick
- MariaDB
- A IIIF server capable of handling tiled JP2 files without a ton of overhead (e.g.,
  [RAIS](https://github.com/uoregon-libraries/rais-image-server))
- Apache/nginx for authentication as well as proxying to NCA and the IIIF server
- Two running [Open ONI][oni] applications: staging and production.
- An [ONI Agent][agent] (at least v1.8.0) must be set up for each ONI instance
  in order to automate some of the functionality from NCA to ONI. The NCA
  server needs to be able to connect to the ONI Agent, but the agent's ports
  should not be open to any other traffic.
  - In our setup, we have an internal-network-only port for the agents, and
    they run using systemd so that they start on reboot and we can specify
    their settings directly in the systemd unit's environment. The ONI Agent
    README should be sufficient to get this working.

[oni]: <https://github.com/open-oni/open-oni>
[agent]: <https://github.com/open-oni/oni-agent>

**Please note**: The easiest way to get a quick demo / test setup of NCA is via
our Docker configuration / setup, and using the dummy ONI Agent set up in
docker compose builds:

- [compose.yml][compose.yml]
- [Docker][docker-dir]

[compose.yml]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/compose.yml>
[docker-dir]: <https://github.com/uoregon-libraries/newspaper-curation-app/tree/main/docker>

This is great to try things out, or for a quick one-off use of NCA. We don't
recommend using it as is for production, however.

We strongly recommend either crafting your own containerized setup with a mind
to production reliability (which we haven't done) or just running NCA on bare
metal - once the prerequisites above are installed, the rest of NCA is very
easy to get running. If you go this route, you'll still find the docker setup
helpful just in terms of understanding the full stack and configuration.

## Compile

Compilation requires:

- A supported version of [Go](https://golang.org/dl/). Go is only required for
  compilation: its runtime does not need to be installed in production as long
  as you compile on the same architecture your production system has (or change
  the `Makefile` to cross-compile for the targeted architecture).

The easiest way to compile is simply running `make` in the source directory.
This will grab various Go packages the application requires, validate the
current code, and build all the binaries.

A full compilation from a clean repository should take about 15 seconds, though
this can depend on network speed the first time dependencies are pulled from
github. Subsequent compiles generally take under 5 seconds. If that's still
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
dockerized services. Again, see our [Development Guide][dev-guide] for details.

### Security Updates

Every now and then a security issue arises in the Go standard library. It's
rare that something critical shows up, but it isn't a bad idea to regularly
(say every month or so) just grab the latest Go compiler, recompile NCA via
`make`, and push the new binaries to production. This ensures any fixes or
security updates get into your instance of NCA without your having to check on
the NCA project itself, or even watch for Go updates.

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
the container starts up. If you're doing development and break the automatic
migration, just run `migrate-database` inside the web container.
