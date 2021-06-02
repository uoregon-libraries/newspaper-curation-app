---
title: Users
weight: 30
description: Creating a new NCA administrative user
---

Newspaper Curation App - User Setup
===

First-time Setup
---

Once the applications are installed and configured, start the NCA server in debug mode:

    ./bin/server -c ./settings --debug

This lets you fake an admin login via `http://your.site/users?debuguser=admin`.
You can then set up other users as necessary.  Once you have Apache set up to
do the authentication, you should never run in debug mode on production servers.

For development use, `docker-compose.override.yml-example` is already set up to
run in debug mode.  Assuming you follow the
[development guide's instructions](/contributing/dev-guide), you won't have to
do anything special to fake your login.
