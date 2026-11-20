---
title: Developer's Guide
weight: 10
description: Developing on NCA
---

Development happens entirely in containers: NCA itself (web server, job
runner, and one-off commands) plus everything it depends on: MariaDB, RAIS
(IIIF), SFTPGo, and two complete Open ONI stacks (staging and production) with
their ONI Agents. Your checkout is mounted into the NCA containers, which
recompile from it whenever they start, so you edit code on your host and
restart a container to see the change.

The same compose setup is meant to run production eventually, so dev and
production differ only in configuration (`.env`) and a small override file.

## Requirements

- [Podman](https://podman.io/) and
  [podman-compose](https://github.com/containers/podman-compose). The stack
  relies on podman features (rootless containers and the `k8s-file` log
  driver), so Docker won't work.
- A supported version of [Go](https://golang.org/dl/), to build the tools
  that run the dev and test stacks (`make tools`). It's also handy for editor
  tooling, running `make validate` / `make test` outside a container, etc. NCA
  itself is compiled in its containers.

## Environment Setup

Get the code:

```bash
git clone git@github.com:uoregon-libraries/newspaper-curation-app.git nca
cd nca
```

You don't have to call the directory "nca", but the docs assume you did.

### Configure

```bash
cp .env.example .env
cp compose.override.example.yml compose.override.yml
```

- `.env` holds **all** configuration: URLs, credentials, which optional
  services run, and so on. Its defaults are meant for dev, so you may not need
  to change anything, but read through it. If you browse from another machine
  (e.g., NCA runs on a VM), set `PUBLIC_HOST` to the name or IP you browse to.
  Public URLs are built from pieces (`PUBLIC_SCHEME`, `PUBLIC_HOST`,
  `PUBLIC_PORT`, `ONI_PROD_PORT`, etc.), and the override publishes services
  on the same ports, so a port collision is a one-line change in `.env`. NCA
  itself lives under a path (`NCA_URL_PATH`, `/nca` by default), so links
  that don't account for one break in dev, not just in production setups that
  use a path.
- `compose.override.yml` is only for *structural* changes: published ports,
  host directories for volumes, and the dev build target. Don't put
  environment variables in it; if `.env` doesn't expose something you need,
  add it to `compose.yml` and `.env.example`.

Variables exported in your shell silently override `.env`, so watch out for
collisions (e.g., a `DB_PASSWORD` from some other project).

### Start the stack

```bash
make tools
bin/ncadev reset
```

`make tools` builds `bin/ncadev`, which runs the dev stack, and `bin/ncatest`,
which runs the [test stack](#test-data). Rebuild them when they change (`make
tools` only rebuilds what's out of date). They work from anywhere in the
checkout, but the docs assume you're in the NCA root.

`bin/ncadev reset` creates the host directories the stack needs (wherever your
override binds volumes, `DEV-VOLS` in the example, and the log directory),
builds the images (slow the first time), starts everything,
migrates the database, and loads seed data: two MARC org codes, the titles in
`test/sources/marc-xml` (also loaded into both ONIs), and one test user per
NCA role. Run it any time you want a clean database.

Once it's done, with the default ports:

- NCA: <http://localhost:3333/nca>. Dev runs in debug mode
  (`NCA_DEBUG=true`), so you pick who you are with a query parameter: e.g.,
  <http://localhost:3333/nca?debuguser=sysop>. `reset` creates one user per
  role, named for the role without spaces (e.g., "sitemanager",
  "issuecurator").
- ONI production: <http://localhost:8080>, and staging:
  <http://localhost:8082>
- SFTPGo's web admin: <http://localhost:8081> (login "admin", password
  "password")

The first start can take ten minutes or more while NCA caches issue data from
the live site (`NEWS_WEBROOT`). The example override keeps the cache on your
host, in `DEV-VOLS/issue-cache`, so this only happens once.

`NEWS_WEBROOT` has to point at a real ONI site with NCA's JSON endpoints
patched in (see [Services][services]), which is why dev uses Oregon's live
site instead of the local ONI.

[services]: <{{% ref "/setup/services" %}}>

### Day to day

`bin/ncadev help` lists everything it can do, but the key commands are:

- `up`: start the stack, or apply changes to it
- `compose ...`: compose with your configured profiles, e.g., `bin/ncadev
  compose ps` or `bin/ncadev compose restart web`. `scripts/compose` does the
  same, and either one is better than a bare `podman compose` (see
  [Profiles](#profiles)).
- `cli <command> [args]`: run a one-off command in a throwaway NCA container,
  e.g., `bin/ncadev cli queue-batches` or `bin/ncadev cli bash`. NCA binaries
  don't need `-c`; the container sets `NCA_CONFIG`.
- `sql`: a MariaDB console on NCA's database, or `bin/ncadev sql -e "..."`
- `reset`: described above
- `prep`: `reset`, then load the test issues (see [Test Data](#test-data))
- `workonce`: stop the job runner service and run jobs in the foreground,
  verbosely, until there are none left
- `curate`, `review`, and friends: stand in for a human at each step of the
  workflow (see [Testing][testing])

Each service logs to its own file in `logs/` (`web.log`, `workers.log`,
etc.), so `tail -f logs/web.log logs/workers.log` is the easiest way to see
what NCA is doing. `podman compose logs` works too.

### Edit, compile, test

The NCA containers run your checkout, and rebuild whatever binary they run
each time they start. After changing Go code:

```bash
bin/ncadev compose restart web workers
```

That usually takes a couple of seconds. Template and static file changes
don't even need a restart, since they're read straight from your checkout.

`bin/ncadev cli` also rebuilds the binary it's asked to run, so `bin/ncadev cli
queue-batches` always runs your current code.

The containers' compiled binaries and Go's caches live in volumes, not your
checkout, so they never mix with anything you build on your host.

Run `make validate` and `make test` before you commit: CI does. If you don't
have Go locally, `bin/ncadev cli make validate test` works.

### Profiles

Optional services are grouped into compose profiles, enabled by
`COMPOSE_PROFILES` in `.env`:

- `local-dev`: the staging and production ONI stacks and their agents
- `sftpgo`: SFTPGo
- `tools`: the `cli` service for one-off commands, which `bin/ncadev cli`
  uses

podman-compose before 1.6 ignores `COMPOSE_PROFILES`, which silently leaves
those services out of `up`, `down`, `ps`, etc. `scripts/compose` and
`bin/ncadev` pass each profile as a `--profile` flag, so use one of them
instead of `podman compose`.

### Volumes and files

NCA's workflow tree (uploads, page review, the workflow itself, batches, etc.)
is the `news` volume. The example override binds it to `DEV-VOLS/news`, so you
can look at and change files directly, and the issue cache to
`DEV-VOLS/issue-cache`. git ignores `DEV-VOLS`. You can bind them somewhere
else: `bin/ncadev` reads the paths from your override.

Under rootless podman, files containers write as root are owned by you on the
host. If something ends up owned by another UID (a container writing as a
non-root user), `podman unshare chown -R 0:0 DEV-VOLS/news` makes it yours
again without sudo.

### Snapshots

`bin/ncadev save <name>` saves the workflow tree and every data volume to
`backup/<name>`, and `bin/ncadev restore <name>` puts them back.
This is handy for repeating something from the same starting point. See
[Testing][testing].

[testing]: <{{% ref "testing" %}}>

### Existing dev environments

If you used the old "hybrid" setup (NCA on your host, services in Docker),
replace your `compose.override.yml` with the new example and create `.env`.
Your old `settings` file is no longer used. The volume layout changed, so
start fresh with `bin/ncadev reset`, or `prep`. To keep your issue cache, move
its files into `DEV-VOLS/issue-cache`.

## Test Data

You won't get far without setting up some test issues. NCA has a rudimentary
setup for grabbing issues from a live server and turning them into testable
data for local use. `bin/ncadev prep` loads them into your dev stack.

For repeatable testing, there's a separate test stack, run by `bin/ncatest`
the same way `bin/ncadev` runs the dev stack. It's configured entirely by the
repository, so your `.env` and override can't change how a test behaves, and
testing never touches your dev stack's data.

The processes are detailed in the `test` subdirectory of the NCA project and
explained at a high level on the [Testing][testing] page.

## Coding

All source code lives under `src/` and is broken up by "local" packages.
Everything which compiles into a standalone binary lives in `src/cmd/`.
Comprehensive documentation exists only in the source code, but can be viewed
with `go doc`; e.g.:

```bash
# Read the entire "issuefinder" package's documentation
go doc ./src/issuefinder

# Read the Finder type's documentation
go doc ./src/issuefinder Finder
```

### Validation

`make` will do basic linting and then compile the code if there were no
compiliation / linter errors.

There are a few unit tests which can be executed via `make test`. Coverage is
spotty at best, but some level of sanity-checking does exist. More
comprehensive end-to-end testing is explained in the
[Testing](/contributing/testing) page.

### General Development Notes

- New migrations run automatically when web or workers start. After other
  major changes (e.g., to `compose.override.yml` or `.env`), bring the whole
  stack down and back up: `bin/ncadev down && bin/ncadev up`
- If things seem "weird", bring the whole stack down and back up
- If you suspect a stale build, remove the `dev-bin` volume (with the stack
  down) and the containers will rebuild everything. The `go-cache` volume
  holds Go's build and module caches, and can be removed the same way.
