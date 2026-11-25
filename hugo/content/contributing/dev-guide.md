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
  [podman-compose](https://github.com/containers/podman-compose) 1.6.0 or
  newer (older versions ignore `COMPOSE_PROFILES`; see [Profiles](#profiles)).
  The scripts (`scripts/reset`, `scripts/savepoint`, and the test stack's)
  call podman directly, and rely on rootless podman to keep the files
  containers write on your host owned by you, so Docker isn't supported.
- Optionally, a supported version of [Go](https://golang.org/dl/), for editor
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
mkdir -p DEV-VOLS/news DEV-VOLS/batches DEV-VOLS/issue-cache
podman compose up -d
```

The example override binds volumes to directories in `DEV-VOLS`, which have
to exist before the stack starts: podman won't create them. The first `up`
builds the images, which is slow. NCA migrates its database whenever it
starts.

Once it's up, with `.env.example`'s ports:

- NCA: <http://localhost:3333/nca>. Dev runs in debug mode
  (`NCA_DEBUG=true`), so you pick who you are with a query parameter: e.g.,
  <http://localhost:3333/nca?debuguser=sysop>.
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

### Seed data

A new database has one user, "sysop", and nothing else. Seed data makes NCA
much more useful to poke at:

```bash
scripts/reset
podman compose up -d
```

`scripts/reset` **deletes all of the stack's data** (NCA's database, both
ONIs' databases and Solr indexes, SFTPGo's data, and the workflow tree), then
loads seed data: two MARC org codes, one user per NCA role (named for the role
without spaces, e.g., "sitemanager" or "issuecurator"), and the titles in
`test/sources/marc-xml`, which it loads into both ONIs, too. With `--issues`,
it loads the test issues as well (see [Test Data](#test-data)). It keeps Go's
caches and the issue cache, asks before deleting anything, and leaves NCA
stopped, hence the `up`.

### Day to day

It's all plain compose, from the NCA root:

- `podman compose up -d`: start the stack, or apply changes to it
- `podman compose down`, `podman compose ps`, `podman compose restart web`,
  etc.
- `podman compose --profile tools run --rm cli <command> [args]`: run a
  one-off command in a throwaway NCA container, e.g., `... cli queue-batches`
  or `... cli bash`. NCA binaries don't need `-c`; the container sets
  `NCA_CONFIG`. Add `--no-deps` to keep it from starting services you've
  stopped, and `-T` if you're piping its output.
- `podman compose exec db sh -c 'MYSQL_PWD=$MARIADB_PASSWORD exec mariadb
  -u"$MARIADB_USER" "$MARIADB_DATABASE"'`: a MariaDB console on NCA's
  database. Or uncomment the db port in your override and use any client.

`run` doesn't build a missing image the way `up` does, so build the `cli`
image before you first use it, and whenever the Dockerfile changes: `podman
compose --profile tools build cli`.

To watch jobs run in the foreground, stop the job runner service and run one
yourself, which stops once there's nothing left to do:

```bash
podman compose stop workers
podman compose --profile tools run --rm --no-deps cli run-jobs -v --exit-when-done watchall
```

`podman compose logs -f web workers` is the easiest way to see what NCA is
doing. Dev doesn't keep log files: logs go wherever podman's default log
driver puts them (usually journald).

If you type some of these a lot, make shell aliases for them.

### Edit, compile, test

The NCA containers run your checkout, and rebuild whatever binary they run
each time they start. After changing Go code:

```bash
podman compose restart web workers
```

That usually takes a couple of seconds. Template and static file changes
don't even need a restart, since they're read straight from your checkout.

The `cli` container also rebuilds the binary it's asked to run, so `podman
compose --profile tools run --rm cli queue-batches` always runs your current
code.

The containers' compiled binaries and Go's caches live in volumes, not your
checkout, so they never mix with anything you build on your host.

Run `make validate` and `make test` before you commit: CI does. If you don't
have Go locally, run them in the `cli` container: `podman compose --profile
tools run --rm cli make validate test`.

### Profiles

Optional services are grouped into compose profiles, enabled by
`COMPOSE_PROFILES` in `.env`:

- `local-dev`: the staging and production ONI stacks and their agents
- `sftpgo`: SFTPGo
- `tools`: the `cli` service for one-off commands, which you enable with an
  explicit `--profile tools`

podman-compose before 1.6.0 ignores `COMPOSE_PROFILES`, which silently leaves
those services out of `up`, `down`, `ps`, etc. Upgrade it if you have an older
version. (`scripts/compose` works around old versions for servers stuck with
a distro package, but dev doesn't need it.)

### Volumes and files

NCA's workflow tree (uploads, page review, the workflow itself, batches, etc.)
is the `news` volume. The example override binds it to `DEV-VOLS/news`, so you
can look at and change files directly, and the issue cache to
`DEV-VOLS/issue-cache`. git ignores `DEV-VOLS`. You can bind them somewhere
else: change your override, and create the directories. `scripts/reset` and
`scripts/savepoint` ask podman where the volumes are, so they work with any
layout.

Under rootless podman, files containers write as root are owned by you on the
host. If something ends up owned by another UID (a container writing as a
non-root user), `podman unshare chown -R 0:0 DEV-VOLS/news` makes it yours
again without sudo.

### Save points

`scripts/savepoint save <name>` saves the stack's state (the workflow tree and
every data volume) to `backup/<name>`, and `scripts/savepoint restore <name>`
puts it back. This is handy for repeating something from the same starting
point. Both stop the stack, and leave it stopped. If a save is interrupted,
`restore` refuses that save point, rather than restoring part of it.

### Existing dev environments

If you used the old "hybrid" setup (NCA on your host, services in Docker),
replace your `compose.override.yml` with the new example and create `.env`.
Your old `settings` file is no longer used. The volume layout changed, so
start fresh with `scripts/reset` (or `scripts/reset --issues`). To keep your
issue cache, move its files into `DEV-VOLS/issue-cache`.

## Test Data

You won't get far without setting up some test issues. NCA has a rudimentary
setup for grabbing issues from a live server and turning them into testable
data for local use. `scripts/reset --issues` loads them into your dev stack.

For repeatable testing, there's a separate test stack, run with
`test/stack/compose` the same way `podman compose` runs the dev stack, and
scripted test recipes that drive it. It's configured entirely by the
repository, so your `.env` and override can't change how a test behaves, and
testing never touches your dev stack's data.

The processes are detailed in the `test` subdirectory of the NCA project and
explained at a high level on the [Testing][testing] page.

[testing]: <{{% ref "testing" %}}>

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
  stack down and back up: `podman compose down && podman compose up -d`
- If things seem "weird", bring the whole stack down and back up
- If you suspect a stale build, remove the `dev-bin` volume (with the stack
  down) and the containers will rebuild everything. The `go-cache` volume
  holds Go's build and module caches, and can be removed the same way.
