---
title: Developer's Guide
weight: 10
description: Developing on NCA
---

It is assumed developers will use Docker for dependencies outside this
repository: ONI (staging and production, and services for both), database, RAIS
(IIIF server), the ONI Agents (staging and production), and SFTPGo. The rest of
the tools are most easily installed locally, and the NCA binaries themselves
are in fact easier by far to install locally versus building them in an image
when code changes.

## Requirements / Setup

### Local dependencies

- A supported version of [Go](https://golang.org/dl/) (e.g., if 1.16 is the
  latest, you want 1.15 or 1.16)
- [revive](https://github.com/mgechev/revive): `go install github.com/mgechev/revive@latest`
- Install [Docker CE](https://docs.docker.com/install/), which will give you
  the `docker` and `docker compose` commands.

If you choose not to compile on your host machine, you will have a slightly
simpler initial install, but there are a few considerations as you edit and
test the code. See [Not Compiling Locally](/contributing/not-compiling-locally).

## Environment Setup

In all cases you'll need the NCA code repository:

```bash
git clone git@github.com:uoregon-libraries/newspaper-curation-app.git nca
cd nca
```

You don't have to specify "nca" as the destination; I just find it easier to
use than the full name. When reading the documentation, if you don't call it
"nca", make sure you mentally replace references to that directory / app name.

### Hybrid Developer

As mentioned before, Docker is the preferred method *for the external services
only*. The various command-line tools, including NCA, should just be installed
locally.

If you want to go fully local on baremetal, it shouldn't be too difficult, but
you'll need to set up everything yourself, including ONI, and configure
everything manually.

#### Configuration

##### Paths

*All paths in the settings file* need to point to your local filesystem. The
various NCA workflow paths need to point to the `test/fakemount` directory.
Here are some examples:

- `APP_ROOT="/home/jechols/nca"`
- `WORKFLOW_PATH="/home/jechols/nca/test/fakemount/workflow"`
- `METS_XML_TEMPLATE_PATH="/home/jechols/nca/templates/xml/mets.go.html"`

Make sure you edit your settings file and adjust *all paths*, not just those
shown above!

The settings file's "command" paths must match the path on your local system:
`GHOSTSCRIPT`, `OPJ_COMPRESS`, etc. The defaults work in most cases, but you
may need to change things to use a local install of some commands, for
instance, or if you're testing in an environment where your `PATH` won't point
to the installed tools.

##### Docker Services

Start by copying the included `compose.override-hybrid-example.yml` file to
`compose.override.yml`. The example has everything needed to run the full stack
locally with minimal fuss.

`compose.override.yml` must expose RAIS ("iiif"), mysql, both oni-agents, and
SFTPGo to the local server via "ports" declarations, and settings need to
reflect these values. It is also useful to expose the oni web services for
testing that NCA is sending the batches to the right ONI instance.

For example, if you use the hybrid compose override values as-is, the critical
settings would look like this:

```
IIIF_BASE_URL=http://localhost:12415/images/iiif
STAGING_NEWS_WEBROOT="http://localhost:8082"
DB_HOST="127.0.0.1"
DB_PORT=3306
DB_USER="nca"
DB_PASSWORD="nca"
DB_DATABASE="nca"
SFTPGO_API_URL="http://localhost:8081/api/v2"
STAGING_AGENT="localhost:2223"
PRODUCTION_AGENT="localhost:2222"
```

Unfortunately, `NEWS_WEBROOT` needs to be set to a running ONI instance that
has the JSON endpoints patched, which isn't in a vanilla ONI instance. NCA
requires those APIs to pull live issue data, and will not run if you try to
point to a vanilla ONI setup.

##### NCA Web Server

`WEBROOT` should just be localhost and whatever port you want. e.g.,
`WEBROOT="http://localhost:3333"`. The port must reflect the port NCA listens
to, as configured in the `BIND_ADDRESS`.

#### Local Development Aliases

A handy script, `scripts/localdev.sh`, has been provided for easier development
and testing. Using it via `source` will expose several useful functions for
easing a more local development environment. Docker is still expected for the
various external services, but the NCA applications will be completely local.

For this to work, however:

- You should read and understand the Docker image definitions
- You must install the command-line tools locally:
  - poppler utils
  - openjpeg
  - Graphics Magick
  - GhostScript
- You should have a solid understanding of how NCA works: which binaries do
  what, the overall workflow both at a high-level and a technical level, etc.
- You should have a decent understanding of bash so you can read through and
  understand how to use `localdev.sh`.
- You must be comfortable working with docker on the command line.

For development, you will need to know about the following functions exposed by
the script:

- `resetdb` initializes the database to prepare for NCA development from a
  "clean slate":
  - Deletes the stack, including database volumes
  - Starts up key services (*db, iiif, sftpgo, and oni services*)
  - Once the database is ready, runs the DB migrations and ingests basic seed
    data that helps our test data repo.
- `migrate` can be run standalone if you don't have seed data and just need to
  get the database migrations run
- `server` prepares, compiles, and runs the HTTP server:
  - Starts up the necessary pieces of the docker stack (see above)
  - Provisions a valid SFTPGo key in your `settings` file
  - Compiles the HTTP server in case any changes have been made to the source
  - Runs the server in debug mode
- `workers` prepares and starts the job runner:
  - Starts up key docker services (see above)
  - Compiles `run-jobs` in case the source code changed
  - Runs the standard "watchall" subcommand for the job runner

These functions are simply added to your bash environment the moment you
`source scripts/localdev.sh`, meaning you can simply type `server` and the
server will start up.

The script exposes a lot of other functions developers generally won't need.
Some are only meant for use in testing, and some are mostly for use by the
script itself and won't be documented. Reading through the script may still be
useful to better understand how it works in case you want to run some commands
manually.

### Docker With Local Compilation

**This is not recommended**, because it can be easy to get things "out of
sync", such as when your host system has a different architecture from the
docker image (in which case the compiled binaries won't run) or the compiled
binaries aren't mounted inside the container properly (in which case you could
be running a different version of the code than you're editing).

This approach also doesn't match the recommended production setup, where docker
isn't used for any of the stack.

However, this approach can be easier than the "hybrid" approach above if you
don't want to deal with all the command-line tool installs (openjpeg, poppler,
graphicsmagick, ghostscript) and figuring out the various docker settings to
expose the services NCA needs (database, sftpgo, oni-agent, IIIF).

#### Copy docker configuration

    cp compose.override-example.yml compose.override.yml

You'll have to figure out setting up binary mounts on your own, as the example
file is built in a more "production-like" approach, where everything is built
and run in a container.

#### Compile

```bash
make
```

Binaries have to be built before starting up docker if you are mounting them
into the container.

#### Get all images

```bash
docker compose build
docker compose pull
```

Building the images will take a little while. Grab some coffee.

Note that once everything has been built, further builds will be quick as
docker will cache the expensive operations (updating the `dnf` cache,
downloading and installing dependencies, etc) and only update what has changed
(e.g., NCA source code).

#### Start the stack

Run `docker compose up`, and all applications will start, exposing ports to
your host however they're defined. Note that on the first run it will take a
while to respond as the system is caching all known issues - including those on
the defined live site.

#### Edit + Compile + Test Loop

Here's a nice shortcut one can use to speed up the process since, unlike PHP,
this project requires compilation before it starts up:

```bash
alias dc='docker compose'
make fast
dc restart web proxy workers
dc logs -f web proxy workers
```

The alias just makes it easier to work with docker in general, and can be put
into a `.bash_aliases` file or similar.

### 100% Docker

This is not recommended for development as you'll have to rebuild the docker
images (or enter the containers to rebuild the binaries) every time code
changes. Most of the time this approach will look a lot like the prior
approach, but you won't be able to just run "make" locally, and you won't mount
your binaries into the container.

Generally you'd go this route for production, where you want the docker image
to be immutable and self-contained. But that's rarely what you want in
development.

Once again, see [Not Compiling Locally](/contributing/not-compiling-locally)
for details on doing dev this way.

## Test Data

You won't get far without setting up some test issues. NCA has a rudimentary
setup for grabbing issues from a live server and turning them into testable
data for local use.

Note that the test scripts/recipes assume devs are using the "hybrid" approach
to development.

The processes are detailed in the `test` subdirectory of the NCA project and
explained at a high level on the [Testing](/contributing/testing) page.

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

- If you make a database schema change (e.g., a new migration), or other major
  changes (e.g., changing your `compose.override.yml` file), you should
  bring the whole stack down and back up
- If things seem "weird", bring the whole stack down and back up
- Only run `make fast` for quick test loops, as it skips static analysis
  validations like code formatting and linting
- Run `make clean` if you don't trust what you're seeing; it'll remove all
  cached compiler output
- Run `make distclean` if you want to delete all cached / downloaded
  dependencies. This should rarely be necessary, if ever.
