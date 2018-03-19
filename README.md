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
through to the batching phase, fixing the batch requires low-level database
fixing and running a command-line utility.

In general, there are undocumented problems which can happen out of the
application's scope, and which can only be fixed by manual intervention due to
features we haven't had time to build and/or general human error inherent in
publisher-uploaded PDFs and scanned+OCRed historic titles.

Ye be warned.

Project
---

This project consists of various scripts for converting
[born-digital](https://en.wikipedia.org/wiki/Born-digital) PDF newspapers, as
well as scanned newspapers, into a one-batch
[bag](https://en.wikipedia.org/wiki/BagIt) which can be ingested into
[ONI](https://github.com/open-oni/open-oni) and
[chronam](https://github.com/LibraryOfCongress/chronam).  See our other
repositories for the legacy suite.  Actually, don't, unless you want a history
lesson.  They're pretty awful:

- [Back-end python tools](https://github.com/uoregon-libraries/pdf-to-chronam)
  - This has been completely deprecated.  YAY!
- [Front-end PHP app](https://github.com/uoregon-libraries/pdf-to-chronam-admin)
  - This still has a few necessary pieces of the project.  BOO!

*Apologies*: this toolsuite was built to meet our needs.  It's very likely some
of our assumptions won't work for everybody, and it's certain that there are
pieces of the suite which need better documentation.

Please refer to the
[wiki](https://github.com/uoregon-libraries/newspaper-curation-app/wiki) for
detailed documentation.
