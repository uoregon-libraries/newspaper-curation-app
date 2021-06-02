---
title: Glossary
weight: 10
description: NCA Glossary
---

## NDNP / Newspaper Catalogers

These are terms I'm trying to use properly, but which I sometimes still mess
up.  Don't let that happen to you!

- **Title**: A distinct newspaper title, such as *The Daily Prophet*.  All
  titles will have a unique LCCN.
- **LCCN**: Library of Congress Control Number:
  <https://en.wikipedia.org/wiki/Library_of_Congress_Control_Number>.  In terms
  of NCA (and ONI and chronam), this uniquely identifies a newspaper title.
- **Issue**: A single published issue of a newspaper title, such as the April
  20th edition of *The Daily Prophet*.  Github users beware, it's easy to hear
  "I need you to fix an issue" and stare blankly before realizing what was
  meant.

## Made-up

Terms I've made up which may be important for developers and users:

- **Issue Key**: This is a combination of an LCCN and optional date elements,
  used for finding / selecting a group of issues in bulk.  This concept is used
  by tools such as the issue finder and the bulk upload queue tool.  An issue
  key's format is `LCCN/YYYYMMDDEE`:
  - `LCCN` is required
  - `/YYYY` is optional, but if present must be a four-digit year
  - `MM` is optional, but if present must be a two-digit month
  - `DD` is optional, but if present must be a two-digit day of the month
  - `EE` is optional, but if present must be a two-digit edition number
  - Each optional part must have all other optional pieces before it; e.g., you
    can't specify `LCCN/MM`
- **MOC** or **MARC Org Code**: Not made up by me, but the short form, "MOC",
  may be seen a lot due to my laziness.  This is supposed to be the MARC
  Organization Code designating the awardee of a batch of newspaper issues.
  However, in chronam and ONI, this determines the image attribution line,
  e.g., "Image provided by: Dallas Public Library; Dallas, OR".  This has lead
  to a necessary evil of misusing MOC in order to provide attribution.
  Currently ONI and chronam don't provide a way to separate image attribution
  from the awardee.  For anything that isn't awarded by the Library of
  Congress, technically there is no awardee, but the software can't actually
  accommodate a batch that doesn't have this set to *something*.
