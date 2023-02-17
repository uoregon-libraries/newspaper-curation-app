---
title: Developer's Guide
weight: 10
description: Developing on NCA
---

It is assumed developers will use Docker for the stack, other than the
relatively simple process of compiling binaries.

## Requirements / Setup

### Local compilation (recommended)

- A supported version of [Go](https://golang.org/dl/) (e.g., if 1.16 is the
  latest, you want 1.15 or 1.16)
- [revive](https://github.com/mgechev/revive): `go install github.com/mgechev/revive@latest`
- Set up your `GOPATH`: https://golang.org/doc/code.html#GOPATH
  - Add `$GOPATH/bin` to your path

If you choose not to compile on your host machine, you will have a slightly
simpler install, but there are a few considerations.  See
[Not Compiling Locally](/contributing/not-compiling-locally).

### Docker

Install [Docker CE](https://docs.docker.com/install/), which will give you the
`docker` and `docker compose` commands.

As mentioned before, Docker is the preferred method for development.  Manual
setup instructions would be needlessly complicated to handle installing the
lower-level libraries you'll need, like a very specific version of
poppler-utils, openjpeg with PNG support, and so forth.

If you choose not to use Docker, you're on your own.  Look at the docker files
included in the repository, and the service/deploy files under `rhel7`.  These
are the *only* references for doing a manual installation.

### Application

#### Grab the NCA repository

    git clone git@github.com:uoregon-libraries/newspaper-curation-app.git nca
    cd nca

You don't have to specify "nca" as the destination; I just find it easier to
use than the full name.  When reading the documentation, if you don't call it
"nca", make sure you mentally replace references to that directory / app name.

#### Copy docker configuration

    cp docker-compose.override.yml-example docker-compose.override.yml

The override file specifies useful things like automatically mounting your
local binaries to speed up the edit+compile+test loop, mounting in your local
templates and static files, mapping the proxy service's port, and running in
debug mode.

    cp env-example .env
    vim .env

`.env` sets up default environment variables which `docker compose` commands
will use.  A sample file might look like this:

```bash
APP_URL="https://jechols.uoregon.edu"
NCA_NEWS_WEBROOT="https://oregonnews.uoregon.edu"
```

This would say that all app URLs should begin with
`https://jechols.uoregon.edu` (the default is `localhost`, which is usually
fine for simple dev work), and that the live issues are found on
`https://oregonnews.uoregon.edu`.  The live newspaper server is expected to
have the legacy chronam JSON handlers, as described in
[Services](/setup/services).

#### Compile

    make

Binaries have to be built before starting up docker if you are mounting them
into the container.

#### Get all images

    docker compose build
    docker compose pull

Building the NCA application image will take a long time.  Grab some coffee.
And maybe a nap....

Note that once it's been built, further builds will be quick as docker will
cache the expensive operations (compiling custom versions of poppler and
openjpeg) and only update what has changed (e.g., NCA source code).

#### Start the stack

Run `docker compose up`, and the application will be available at
`$APP_URL/nca`.  Note that on the first run it will take a while to respond as
the system is caching all known issues - including those on the defined live
site.

### Test Data

You won't get far without setting up some test issues.  NCA has a rudimentary
setup for grabbing issues from a live server and turning them into testable
data for local use.

The process is detailed on the [Testing](/contributing/testing) page.

## Coding

All source code lives under `src/` and is broken up by "local" packages.
Everything which compiles into a standalone binary lives in `src/cmd/`.
Comprehensive documentation exists only in the source code, but can be viewed
with `go doc`; e.g.:

    # Read the entire "issuefinder" package's documentation
    go doc ./src/issuefinder

    # Read the Finder type's documentation
    go doc ./src/issuefinder Finder

### Validation

`make` will do basic linting and then compile the code if there were no
compiliation / linter errors.

There are a few unit tests which can be executed via `make test`.  Coverage is
spotty at best, but some level of sanity-checking does exist.  More
comprehensive end-to-end testing is explained in the
[Testing](/contributing/testing) page.

### Edit + Compile + Test Loop

Here's a nice shortcut one can use to speed up the process since, unlike PHP,
this project requires compilation before it starts up:

    alias dc='docker compose'
    make fast
    dc restart web proxy workers
    dc logs -f web proxy workers

The alias just makes it easier to work with docker in general, and can be put
into a `.bash_aliases` file or similar.

### General Development Notes

- If you make a database schema change (e.g., a new migration), or other major
  changes (e.g., changing your `docker-compose.override.yml` file), you should
  bring the whole stack down and back up
- If things seem "weird", bring the whole stack down and back up
- Only run `make fast` for quick test loops, as it skips static analysis
  validations like code formatting and linting
- Run `make clean` if you don't trust what you're seeing; it'll remove all
  cached compiler output
- Run `make distclean` if you want to delete all cached / downloaded
  dependencies.  This should rarely be necessary, if ever.

## Advanced Users

A handy script, `scripts/localdev.sh`, has been provided **for advanced
users**.  Using it via `source` will expose several useful functions for easing
a more local development environment.  Docker is still expected for the IIIF
server and the database, but the NCA applications will be completely local.
This can be a much faster way to do development if you don't mind a more
complicated setup.

For this to work, however:

- You must understand how the Docker image works and replicate it locally.
  This means all the dependencies, like poppler utils and openjpeg, must be
  installed *locally*.  Don't pursue this avenue if you don't know how or
  aren't comfortable locally installing these things.
- You must have a strong understanding of how NCA works: which binaries do
  what, the overall workflow both at a high-level and a technical level, etc.
- You must have a pretty thorough understanding of bash so you can read through
  `localdev.sh` and figure out which commands make sense.  They won't be
  documented very carefully here.
- You must be comfortable working with docker on the command line.

Settings:

- *All paths* need to point to your local filesystem, e.g.,
  `APP_ROOT="/home/jechols/nca"`.
- `WEBROOT="http://localhost:8080"`
- `IIIF_BASE_URL="http://localhost:12415/images/iiif"`
- Paths to commands must match the path on your local system: `GHOSTSCRIPT`,
  `OPJ_COMPRESS`, etc.

Additionally, `docker-compose.override.yml` needs to expose RAIS ("iiif") on
12415 and mysql ("db") on 3306.

The rest is left as an exercise for the reader.  Really, if you made it this
far, and you grasp bash, reading the localdev script should get you the rest of
the way.
