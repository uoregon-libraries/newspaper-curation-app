---
title: Services and Apps
weight: 5
description: The services in the NCA suite
---

You should at least understand everything in this document at a high level
before moving on to the setup/installation documents, as the NCA suite is a set
of tools, not a single application that does it all.

## Overview

NCA has two key services which you'll have to have running in the background at
all times, several binaries you'll need to use occasionally for regular tasks,
and of course the various external services (such as a IIIF server, SFTP
server, MySQL / MariaDB, Apache / nginx, Open ONI and the ONI Agent, etc.).

The docker setup is easy to get running, and extremely useful for learning how
the stack is set up, but it's not something we recommend for production use. It
bundles in two ONI servers and their dependencies (database, solr, and ONI
Agent for both staging and production), there's no monitoring, and the setup
has never been tested in a high-traffic environment.

*Only go the container route if you have the devops chops to build a real,
production-ready setup.*

A bare-metal, manual installation is fairly easy to set up as well. Manual
installations are described in the [installation][installation] documentation,
and can be reverse-engineered by reading the various docker files, especially
if you're on a Redhat / CentOS / RockyLinux setup.

[installation]: <{{% ref "installation" %}}>

**Note**: the repository contains working examples for systemd services and
rsyslog configuration: see the ["deploy" directory on github][deploy-github] to
better understand how you might manage a production installation.

[deploy-github]: <https://github.com/uoregon-libraries/newspaper-curation-app/tree/main/deploy>

## HTTP Server

`server` is the web server which exposes all of NCA's workflow UI. Please
note that, at the moment, this requires Apache sitting in front of the server
for authentication.

Running this is fairly simple once settings are configured:

```bash
/usr/local/nca/server -c /usr/local/nca/settings
```

### Gotcha

**NOTE**: `server` builds a cache of issues and regularly rescans the
filesystem and the live site to keep its cache nearly real-time for almost
instant lookups of issue data. However, building this cache requires the live
site to use the same JSON endpoints chronam uses.

ONI's JSON endpoints were rewritten to use IIIF, so out of the box, ONI isn't
compatible with this cache-building system. The IIIF endpoints supply very
generic information, which didn't give us issue-level information without
performing thousands of additional HTTP requests, so we had to put the old JSON
responses back into our app. If you wish to use this application with an ONI
install, you'll need to do something similar.

The relevant commit links follow:

- [Override IIIF JSON endpoints with previous JSON](https://github.com/uoregon-libraries/oregon-oni/commit/067ab17084d9015996932d2e001226aa18bbcdb6)
- [Fix batch JSON pagination](https://github.com/uoregon-libraries/oregon-oni/commit/0463435615b23058ca1bc2afd8017e7001dc0657)
- [Fix missing route name](https://github.com/uoregon-libraries/oregon-oni/commit/94f84a30abd6ad5a38c8bd932a95297e1a9b1989)

## Job Runner

Queued jobs (such as SFTP issues manually reviewed and queued) will not be
processed until the job runner is executed.

The best way to run jobs is via the "watchall" subcommand:

```bash
./bin/run-jobs -c ./settings watchall
```

This starts the job runner, which will watch all queues and run jobs as they
come in. When invoked this way, the job runner will simply run forever to
ensure jobs are processed whenever there's work to be done.

If you only want to drain all pending jobs and then quit, you can add
`--exit-when-done` to the command.

Finally, there's a subcommand to run a single job and then exit:

```bash
./bin/run-jobs -c ./settings run-one
```

This is primarily a development tool for debugging long pipelines where a
single job seems to be breaking app state, but it can be used to also very
closely monitor exactly which jobs are running in what order, if such a need
should arise.

## Batch Queue

Batch managers can generate batches on demand using the UI, but we also have
the `queue-batches` tool for manually generating every available batch at once.
This can of course be set up to run on cron if so desired.

Execution is simple:

```bash
./bin/queue-batches -c ./settings
```

The job runner will do the rest of the work, eventually putting batches into
your configured `BATCH_OUTPUT_PATH`, syncing to the `BATCH_PRODUCTION_PATH`,
and calling out to the ONI Agent to ingest the batch onto staging.

The tool can be given flags for `--min-batch-size` and `--max-batch-size` in
order to override the standard settings. For instance, our cron job is set to
only create batches when there are several thousand pages ready. It ensures
that if batch managers are out or don't have time to get into the UI, we're
still avoiding a massive backlog of issues waiting to be batched.

## ONI Agent tester

A normal "make" run creates `bin/agent-test`. This is very handy to validate
connectivity to ONI Agents running on your staging and production servers.

In the event of odd batch problems, you can also use this test as a normal
tool: it sends real commands to a running agent so you can do batch loading,
for instance, without dealing with the normal flow of sshing into your ONI
service, activating your Python virtual environments, etc.

A simple use for just checking a job might look like this:

```bash
./bin/agent-test -c ./settings -e staging job-logs 1
```

As this is primarily a testing/debugging tool, it's not going to be
well-documented here, but the source code is very simple, and if you just give
it an invalid command it will tell you what commands *are* valid. e.g., instead
of the command `job-logs`, give it `foo`.

## Bulk Upload Queue

The `bulk-issue-queue` tool allows you to push uploaded issues into the
workflow in bulk. This should *only* be used when you have some other
validation step that happens to the issues of the given type (born digital or
scanned), otherwise you may find a lot of errors that require manual
intervention of issues in the workflow, which is always more costly than
catching problems prior to queueing.

Sample usage:

```bash
./bin/bulk-issue-queue -c ./settings --type scan --key sn12345678
```

Run without arguments for a more full description of options.

## Live File Cleanup

`delete-live-done-issues` is a tool which removes all files NCA tracks after
their corresponding batch has been marked "archived" for four weeks (to ensure
it's safe to remove them). This should be run regularly to prevent disk space
exhaustion - holding onto hundreds of gigs of TIFFs that are backed up outside
NCA, for instance.

## Database Migration

To simplify database table creation / updating, the `migrate-database` binary
is provided. We'll generally mention in the release notes / changelog that a
version requires database migrations and give the example command to run this,
so it isn't typically something you would run on your own, but it's also
harmless if you do run it manually - it won't re-run any database update
scripts that have already run.

## Clean Dead Issues

`remove-dead-issues` is useful to move all "stuck" issues out of NCA and into
the configured problem folder. The original files will be moved, and the full
activity log stored as a text file to help identify how to fix whatever problem
prevented curators (or NCA job runners) from processing an issue.

## Other Tools

You'll find a lot of other tools in `bin` after compiling NCA. Most
have some kind of useful help, so feel free to give them a try, but they won't
be documented in depth. Most are one-offs to help diagnose problems or test
features, and shouldn't be necessary for regular use of this software.

## IIIF Image Server

A IIIF server is not included (and it wouldn't make sense to couple into every
app that needs to show images). However, in order to use NCA to see newspaper
pages, you will need a IIIF server of some kind.

[RAIS](https://github.com/uoregon-libraries/rais-image-server) is the
recommended image server: it's easy to install and run, and it handles JP2s
without any special configuration.

A simple invocation can be done by using the NCA settings file, since
it is compatible with bash, and has all the settings RAIS needs:

    source /path/to/nca/settings
    /path/to/rais-server --address ":12415" \
        --tile-path $WORKFLOW_PATH \
        --iiif-url "$IIIF_BASE_URL" \
        --log-level INFO
