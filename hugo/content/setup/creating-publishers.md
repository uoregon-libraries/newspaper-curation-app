---
title: Onboarding A Publisher
weight: 40
description: Setting up a new NCA toolsuite
---

Creating a publisher in NCA, at least for UO, requires several manual processes
take place:

- Add a title to NCA with sftp credential information.
- If necessary, import the title to your ONI site (for example,
  oregonnews.uoregon.edu).
  - See [Adding Titles](/workflow/technical/adding-titles) for details
- Add a user to the sftp server
  - At UO, we have an internal sftp script at `/usr/local/scripts/addsftpuser.sh`.
  - e.g., `/usr/local/scripts/addsftpuser.sh "newpublisher" "Pas$w0rd"`
- Symlink the sftp server's location so NCA can see it.  NCA's server has a
  helper script at `/usr/local/scripts/add-sftp-symlinks.sh` which lists which
  publishers' directories may need links.
