Newspaper Curation App
===

**Note**: this project should not be considered production-ready unless you
have a developer who can make sense of some of the inner workings.  The
application / suite work, but there are quite a few situations where somebody
needs to really dig to deal with problems.

For instance, a scanner error may require deleting the issue record from the
database, moving the issue's TIFF/PDF files somewhere they can be examined and
fixed by the scanning team, etc.

There are also improvements which need to be made to automate more parts of the
process.  For instance, right now if an issue has errors, but manages to slip
through to the batching phase, fixing the batch requires the use of a somewhat
archaic command-line utility that isn't terribly well-documented.

In general, there are undocumented problems which can happen out of the
application's scope, and which can only be fixed by manual intervention due to
features we haven't had time to build and/or general human error inherent in
publisher-uploaded PDFs and scanned+OCRed historic titles.

**Note 2**: NCA isn't meant as an out-of-the-box solution for anybody but us.
Some of the tools may be generic, but there is no customization of things like
workflow rules, template theming, etc.

Ye be warned.

Project
---

This project consists of various scripts for converting
[born-digital](https://en.wikipedia.org/wiki/Born-digital) PDF newspapers, as
well as scanned newspapers, into a one-batch
[bag](https://en.wikipedia.org/wiki/BagIt) which can be ingested into
[ONI](https://github.com/open-oni/open-oni).

Please refer to
[NCA's online documentation](https://uoregon-libraries.github.io/newspaper-curation-app/) for
detailed documentation.
