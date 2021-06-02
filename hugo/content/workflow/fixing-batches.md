---
title: Fixing Batches
weight: 50
description: Fixing batches after QC failure
---

## Install

Build and run the batch fixer tool.  You'll need the same prerequisites as are
necessary for the rest of the toolsuite, and you'll need to run it on your
production system in order to manipulate batch files.

```bash
git clone https://github.com/uoregon-libraries/newspaper-curation-app.git
cd newspaper-curation-app
make
./bin/batch-fixer -c ./settings
```

The tool is an interactive command-line application.  You can type "help" at
any time to get a full list of commands for your current context.

## Top Menu

### list

The initial context has very few commands, and you likely want to start with "list":

```
$ ../bin/batch-fixer -c ../settings

No batch or issue loaded.  Enter a command:
> list
  - id: 34, status: qc_ready, name: batch_oru_20181128MahoganyNamahageSurroundedByStrawberries_ver01
  - id: 38, status: qc_ready, name: batch_roguerivervalleyirrigationdistrict_20181128BronzeXiangliuTramplingKelp_ver01
```

### load

You load a batch using the "load" command with an id, e.g., `load 34`.  This
puts you into a new context which changes your commands.

## Batch Menu

From there you can type "help" again to get a new list of batch-context
commands.  Not all commands will be explained here, as the in-app help is
likely to be more useful.

### info

Shows you some metadata around the batch.  Useful for ensuring you loaded what
you think you did.

### failqc

If the batch is ready for QC, you can fail it by typing "failqc".  This would
update the batch status as well as removing its files from your batch output
path so that the batch can be regenerated when it's fixed.

### delete

After failing a batch, you have the option to delete it completely.  *This
should only be done if the batch is so broken that removing bad issues
individually is less feasible than manually removing issues from the database!*

All issues will be removed from the batch, but their metadata will remain unchanged otherwise.  If a large number of issues have to be corrected, you'll have to remove them via direct SQL, e.g.:

```sql
UPDATE issues
  SET workflow_step = 'UnfixableMetadataError' AND error = 'Re-OCR all these!'
  WHERE lccn = 'sn88086023' AND workflow_step = 'ReadyForBatching' AND ...;
```

**This is obviously dangerous**.  To reiterate, if it is at all possible,
issues should be removed individually.

### search / list

You can list all issues associated with a batch using "list".  You can also
search for a particular issue using the "search" command with regular
expressions.  For instance, "search date=19[0-6].*" will find any issue that's
got a date of 1900 - 1969.  You can search by lccn, issue key, date, and/or title.  You can combine terms to make searches very refined, e.g.:

    search lccn=sn12.* date=19[0-6].* key=.*02 title=.*blue.*

This would search for issues from 1900-1969 where the key ends in "02" (second
edition issue), the lccn starts with "sn12", and the title contains the word
"blue" somewhere in it.  Search terms are combined via "AND" logic - all terms
you list must match for an issue to be listed.

### load

In batch context, "load" will load an issue by its id.  This will only work if
the issue belongs to the batch you're working on.

## Issues

Again, rely on "help" as much as possible.

### info

This shows details about the loaded issue.

### reject

Flags an issue as having metadata problems which can be fixed in NCA.  This
removes the issue from the batch, deletes its METS file, and puts it back on
the desk of the metadata entry person.

The rejection notes will store whatever you put after "reject".  e.g., "reject
page 19 is mislabeled".

### error

Flags an issue as having metadata problems which *cannot* be fixed in NCA.  This
removes the issue from the batch, deletes its METS file, and flags it for
removal from the system using the "move-errored-issues" tool.

The error notes will store whatever you put after "reject".  e.g., "reject
page 19 and 20 are out of order".

## Load into staging / production / wherever

Rebuild the batch with the issues that still remain:

```bash
### [On the NCA server] ###

cd /usr/local/nca
./requeue-batch -c ./settings --batch-id 29
```

Once it's rebuilt, you can reload into staging if you want to double-check the
batch, or if you're confident the first round of quality control caught all
issues, load to production.  This document won't cover that process.  See
[Batch Manual GoLive](/workflow/batch-manual-golive) for details of loading a
batch into production.
