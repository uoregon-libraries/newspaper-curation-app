---
title: Services and Apps
weight: 5
description: The services in the NCA suite
---

You should at least understand everything in this document at a high level
before moving on to the setup/installation documents, as the NCA suite is a set
of tools, not a single application that does it all.

## Overview

NCA has two key services which you'll have to run, in addition to the rest of
the external services (such as an IIIF server, MySQL / MariaDB, and Apach)

If you're doing a manual installation rather than container-based, you are
strongly advised to look at the docker files - they make it clear precisely how
the stack should be set up.

**Note**: If you do go manual, the repository contains working examples for
RHEL 7 systemd services to start the job runner as well as the workflow http
server: <https://github.com/uoregon-libraries/newspaper-curation-app/tree/main/rhel7>.
Consider looking at these to better understand how you might manage a
production installation.

## HTTP Server

`server` is the web server which exposes all of NCA's workflow UI.  Please
note that, at the moment, this requires Apache sitting in front of the server
for authentication.

Running this is fairly simple once settings are configured:

    /usr/local/nca/server -c /usr/local/nca/settings --parent-webroot=/odnp-admin

This currently relies on running the
[legacy pdf-to-chronam-admin](https://github.com/uoregon-libraries/pdf-to-chronam-admin)
tool, though we're planning to phase that out eventually.  Again, see the
docker files for examples of how you might set this up.

### Gotcha

**NOTE**: `server` builds a cache of issues and regularly rescans the
filesystem and the live site to keep its cache nearly real-time for almost
instant lookups of issue data.  However, building this cache requires the live
site to use the same JSON endpoints chronam uses.

ONI's JSON endpoints were rewritten to use IIIF, so out of the box, ONI isn't
compatible with this cache-building system.  The IIIF endpoints supply very
generic information, which didn't give us issue-level information without
performing thousands of additional HTTP requests, so we had to put the old JSON
responses back into our app.  If you wish to use this application with an ONI
install, you'll need to do something similar.

The relevant commit links follow:

- [Override IIIF JSON endpoints with previous JSON](https://github.com/uoregon-libraries/oregon-oni/commit/067ab17084d9015996932d2e001226aa18bbcdb6)
- [Fix batch JSON pagination](https://github.com/uoregon-libraries/oregon-oni/commit/0463435615b23058ca1bc2afd8017e7001dc0657)
- [Fix missing route name](https://github.com/uoregon-libraries/oregon-oni/commit/94f84a30abd6ad5a38c8bd932a95297e1a9b1989)

## Job Runner

Queued jobs (such as SFTP issues manually reviewed and queued) will not be
processed until the job runner is executed.  You will want to ensure at least
one process is watching each type of job, and one process is watching the page
review folder for issues ready to be queued up for derivatives.

A simple approach to run everything needed is as follows:

    ./bin/run-jobs -c ./settings watchall

This starts the job runner, which will watch all queues and run jobs as they
come in. When invoked this way, the job runner will simply run forever to
ensure jobs are processed whenever there's work to be done.

If you only want to drain all pending jobs and then quit, you can add
`--exit-when-done` to the command.

Finally, there's a subcommand to run a single job and then exit:

    ./bin/run-jobs -c ./settings run-one

This is primarily a development tool for debugging long pipelines where a
single job seems to be breaking app state, but it can be used to also very
closely monitor exactly which jobs are running in what order, if such a need
should arise.

## Batch Queue

The queue-batches tool is currently run manually.  Until more of the batch
ingest can be automated, it is safest to require somebody to manually watch the
process which tries to gather up issues into a batch.  This can of course be
set up to run on cron if so desired.

Execution is simple:

    ./bin/queue-batches -c ./settings

The job runner will do the rest of the work, eventually putting batches into
your configured `BATCH_OUTPUT_PATH`.  You'll know they're ready once batch
folders have been named `batch_*`, as the names are always `.wip*` until the
batch is safe to load into a staging environment.

**Note** that even when batches are ready for staging, there is still a
potentially slow job to be done generating the bag manifest and other
[BagIt](https://en.wikipedia.org/wiki/BagIt) tag files.  These files aren't
necessary for ingest, and serve primarily to help detect data degradation, but
the batch should not be considered production-ready until that job is done.  At
the moment the only way to detect that job's completion is either looking at
the jobs table directly or else checking for a complete and valid
"tagmanifest-sha256.txt" in the batch root directory.

## Bulk Upload Queue

The `bulk-issue-queue` tool allows you to push uploaded issues into the
workflow in bulk.  This should *only* be used when you have some other
validation step that happens to the issues of the given type (born digital or
scanned), otherwise you may find a lot of errors that require manual
intervention of issues in the workflow, which is always more costly than
catching problems prior to queueing.

Sample usage:

    ./bin/bulk-issue-queue -c ./settings --type scan --key sn12345678

Run without arguments for a more full description of options

## Other Tools

You'll find a lot of other tools in `bin` after compiling NCA.  Most
have some kind of useful help, so feel free to give them a try, but they won't
be documented in depth.  Most are one-offs to help diagnose problems or test
features, and shouldn't be necessary for regular use of this software.

## IIIF Image Server

An IIIF server is not included (and it wouldn't make sense to couple into every
app that needs to show images).  However, in order to use NCA to see newspaper
pages, you will need an IIIF server of some kind.

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
