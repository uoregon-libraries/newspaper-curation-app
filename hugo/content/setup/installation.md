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

**Please note**: NCA has a container setup (podman compose) that runs the
whole suite, including a pair of demo ONI stacks for NCA to load batches into.
It's how we develop NCA, and it's built to run production as well, though we
haven't moved our own production systems to it yet. See
[Containers][containers] for details, and the [compose.yml][compose.yml] and
[docker directory][docker-dir] on GitHub.

[containers]: <{{% ref "containers" %}}>
[compose.yml]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/compose.yml>
[docker-dir]: <https://github.com/uoregon-libraries/newspaper-curation-app/tree/main/docker>

The rest of this page covers a manual (bare metal) installation. Even if you
go that route, the container setup is helpful for understanding the full stack
and its configuration.

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

In the container setup, the web and workers services migrate the database
automatically whenever they start.
