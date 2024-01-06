---
title: Handling Page Review Problems
weight: 30
description: Dealing with problems created when issues are in the "page review" area of NCA
---

The "page review" location is one of the most dangerous in the application due
to the requirement that people manually edit and rename files.  There are
potentially a *lot* of difficult problems to manage here.

## Manual Deletion

If issues got to the page-review step when they shouldn't have, the only option
is to remove them.  **Make sure you do this right**.  There is a tool for this
process, and manually deleting issues will cause you *pain* (see below).

To use the tool, build NCA with `make`, and run `./bin/page-review-issue-fixer
-c ./settings --key ...`.  You can specify multiple issue keys or even a file
full of issue keys.  Use `--help` for a complete explanation.

Once this tool runs, if all goes well the issues will get queued up and moved
to the configured error location.  From there you can do whatever you like with
the issues, as described in [Fixing Flagged Workflow Issues][1].

[1]: <{{% ref "fixing-flagged-workflow-issues" %}}>

## Accidental Deletion

Sometimes a helpful curator deletes the issues manually, not being aware this
shouldn't happen.  On these occasions, manual cleanup is required, and it gets
very ugly.

### Identify the deleted issues

If you don't know offhand what's been deleted, but you've seen log errors about
the page review location, this might help.

```bash
cat /var/log/nca-* | grep " - ERROR - " | \
    sed 's|^.* - ERROR - ||' | \
    sed 's|^.* \(/mnt/news/.*\): no such file or directory$|\1|' | \
    sort | uniq -c | sort -n
```

This will find all error logs that are due to file or directory being missing.
It will likely catch other problems than just page-review deletions, but those
can be useful as well.  Just note that things like NFS drop can result in
occasional one-offs.  You really want to look for systemic, repetitive errors.
In our case, we saw almost 400 errors per issue because NCA scans the
filesystem every few minutes.

(If you're using a smart logging system like logstash, you can probably
identify logs more easily, but you'll likely need to split logs up to strip off
the changing bits so the unique issues can be aggregated and counted)

The database ids are the last number in an issue's path.  e.g.,
`/mnt/news/page-review/sn99063854-1925122501-9971` has a database id of 9971.

### Fix the data

First, **verify** that the issues are in fact not on the filesystem.  If you
see errors about database id 9971, you can query its location:

```sql
SELECT location FROM issues WHERE id=9971;
```

If that location exists, the problem you have is *not* what is described here.

Assuming the locations are indeed deleted, gather up all your database ids.  If
you had errors with ids 9971, 9975, and 9990:

```sql
UPDATE issues SET
    error = 'manually deleted from page-review step',
    workflow_step = 'UnfixableMetadataError',
    location='',
    ignored=1
WHERE id in (9971, 9975, 9990);
```

The database is fixed!

### Delete backups?

If the issues are completely ruined and the backups are known to be bad, you
should delete them.  Get their locations from the database:

```sql
SELECT backup_location FROM issues WHERE id IN (9971, 9975, 9990);
```

Remove these locations from disk.
