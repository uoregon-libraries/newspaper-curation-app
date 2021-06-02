---
title: "Uploads: Folder and File Specs"
weight: 20
description: Detailed specifications for uploaded issues' files
---

## Uploads: Folder and File Specs

Publishers (or in-house scanners) who upload issues must adhere to very strict
structures for issue organization.

### Born-Digital

The folder structure tells us the newspaper title and issue date.  e.g.,
`/mnt/news/sftp/sn12345678/2018-01-02` would mean the January 2nd, 2018 edition
of title `sn12345678`.  Note that the "sftp directory" can be used instead
of the LCCN if they differ (for historical reasons - the legacy app's
born-digital and scan processors were very separated).

The issue should contain PDFs and nothing else.  Publishers should never upload
tertiary files.  Ideally, publishers should upload one PDF for the entire
issue, with pages in the order they wish to see on the ONI site, as that
reduces (or eliminates) the need to have anybody reviewing these issues' pages.

Some publishers may be unable (or unwilling) to comply with the aforementioned
folder structure.  It may be necessary to build a custom pre-processor that
takes uploaded files and restructures them for the application.  In some cases,
there may even need to be human intervention to determine the right issue
folder name.

**Note**: Currently NCA puts all born-digital issues into a single MARC Org
Code when generating batches (determined by the `PDF_BATCH_MARC_ORG_CODE`
setting), and doesn't support having multiple editions of a single date.

### Scans

The folder structure tells us the same information as with born-digital
uploads, but also includes a MARC Org Code.  e.g.,
`/mnt/news/scans/oru/sn12345678/2018-01-02` would be the January 2nd, 2018
issue of title `sn12345678`, and would be assigned the `oru` awardee when
batched.

Additionally, issues can have multiple editions by adding `_dd` to the issue
folder name; e.g., `.../2018-01-02_03` would designate the third edition
for January 2nd, 2018.

These issues should contain one TIFF per page and one PDF per page.  The PDF
should contain the TIFF's image with OCR information as an application like
Abbyy produces.

To conform closely to the NDNP spec, the TIFF files should be at least 300dpi
and the PDFs should contain a 150dpi JPEG image encoded at about a quality of
40 (or "medium").
