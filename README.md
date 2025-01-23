# Newspaper Curation App

**Note**: this project isn't non-tech-user-friendly! You will need to have at
least a part-time developer who can make sense of some of the inner workings.
The application / suite work, but there are quite a few situations where
somebody needs to really dig to deal with problems.

For instance, external integrations can be finicky, particularly when trying to
automate Open ONI commands. If your ONI instance fails to ingest a batch that
NCA sends it, you likely need somebody to get their hands dirty in order to
diagnose and fix the problem. This may mean looking through the codebases of
these projects, doing manual database cleanup, etc.

More generally, there are a wide variety of problems which can happen out of
the application's scope, and which can only be fixed by manual intervention,
usually do to human errors which are inherent in publisher-uploaded PDFs and
scanned+OCRed historic titles.

**Note 2**: NCA isn't meant as a customizable out-of-the-box solution. Most of
the tools are reusable by anybody curating newspapers, but there is no way to
apply your own branding, set up complex workflow rules, etc.

Ye be warned.

## Project

This project consists of various scripts for converting [born-digital][1] PDF
newspapers, as well as scanned newspapers, into a one-batch [bag][2] which can
be ingested into [ONI][3].

Please refer to [NCA's online documentation][4] for detailed documentation.
Please note that our documentation is for the *latest stable release*, not
necessarily what's in our `main` branch.

If you're looking for bleeding edge documentation, you can either browse the
[hugo documentation directly in our source][5], or check out the repo and use
hugo directly to build and host docs yourself. The ["Contributing to
Documentation" document][6] describes how to generate docs manually.

[1]: <https://en.wikipedia.org/wiki/Born-digital>
[2]: <https://en.wikipedia.org/wiki/BagIt>
[3]: <https://github.com/open-oni/open-oni>
[4]: <https://uoregon-libraries.github.io/newspaper-curation-app/>
[5]: <https://github.com/uoregon-libraries/newspaper-curation-app/tree/main/hugo/content>
[6]: <https://uoregon-libraries.github.io/newspaper-curation-app/contributing/documentation/>
