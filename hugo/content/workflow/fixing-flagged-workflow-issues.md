---
title: Fixing Flagged Workflow Issues
weight: 40
description: Fixing issues which have errors NCA cannot fix
---

This refers to issues which were already queued from their uploaded location,
had derivatives generated, and were ready for metadata entry.

The metadata entry person flags errors to say essentially, "NCA's UI cannot fix
the problems on this issue".  We have seen a variety of problems like this:

- The PDF or JP2 image derivatives are corrupt in some way, even though the
  tools which generated them seemed to run without errors
- The pages are out of order - somebody reordered pages incorrectly, and the
  issue now has to be manually pulled, fixed, and re-inserted into the workflow
- The issue is incorrect in some other way, and wasn't caught when queueing
  from the uploads area (e.g., a publisher uploaded two issues in the same
  location, pages were missing from an upload, etc.)

Most errors can be caught prior to queueing an issue for processing, so it is
very important that curators be aware of the additional cost of having to fix
issues that are incorrect after they've gotten into the workflow.

## Identifying bad issues

NCA now provides a place for privileged users to process "unfixable" errors.
Anybody with the "Issue Manager" role can see a tab in the Workflow section of
the application labeled "Unfixable Errors".  In this tab, issue managers can
claim and then process these issues, choosing to return them back to NCA if
they were flagged incorrectly, or move them to a configured error location
(`ERRORED_ISSUES_PATH` in the settings file).

When moved to the error location, the issues will be put into a directory based
on the current month so that they're somewhat organized without having so many
subdirectories as to make the process more painful than necessary.

Within the month subdirectory, issues will be identifiable by their LCCN, date,
edition, and database id in the same way they existed in the workflow location.
This will look something like `sn96088087-2010041901-1`.  Under that directory
you will find `content` and `derivatives`.  The derivatives are preserved just
in case debugging is necessary (e.g., if a JP2 is broken, but the source PDF
seems fine).  The content directory will contain the source files, including
original uploads in the case of publishers' sftp-delivered files, in an archive
called `original.tar`.

Additionally, a file called `actions.txt` will be present and describe all
actions taken on the issue along with any comments written by curators and
reviewers.

## Fixing removed issues

This is a much more difficult problem to solve, because of the wide variety of
errors that can occur.  Fixing problems will typically require a manual
examination of each issue that was removed.  The `actions.txt` file should help
understand what caused the issue to be removed, but that won't necessarily help
fix the problem.

There are cases where the only option is to delete the issue entirely and
accept that it will not be able to be a part of your archive.  This can happen
if the publisher uploaded the wrong issue and no longer has access to the
original files, or if scanned papers' TIFFs were corrupt and the original paper
is no longer available.

## Putting issues back into the workflow

If an issue can be fixed and put back into the workflow, there are typically
two options:

- Pretend it's a new upload and start right at the beginning.  Derivative files
  must be removed, and the uploads must conform to the file and folder specs as
  defined in our [file/folder upload specs](/specs/upload-specs).
- Just delete the issue folder and get it re-scanned or re-uploaded from the
  publisher.  In this case, the normal procedures are followed and the issue
  will be completely new for all intents and purposes.

Database manipulation is almost never the right approach once issues have been
moved out of the workflow.  If you believe that you need to manipulate the
database to get an issue back into the workflow, you need to be **100%
certain** you understand the application as well as *the precise meaning of
every field in every table*.

Generally speaking, if database manipulation *is* the correct approach, it
should have been done *instead of* reporting an error and removing the issue.
