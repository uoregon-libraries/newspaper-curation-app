---
title: Users
weight: 30
description: Creating a new NCA administrative user
---

Once the applications are installed and configured, you'll need an admin user.
Start the NCA server in debug mode:

```bash
./bin/server -c ./settings --debug
```

This lets you fake an admin login via `http://your.site/users?debuguser=admin`.
You can then set up other users as necessary. Once you have Apache set up to
do the authentication, you should never run in debug mode on production servers.

For development use, `compose.override.yml-example` is already set up to run in
debug mode. Assuming you follow the [development guide's][dev-guide]
instructions, you won't have to do anything special to fake your login.

[dev-guide]: <{{% ref "/contributing/dev-guide" %}}>
