---
title: Users
weight: 30
description: Creating a new NCA "SysOp"
---

## Create a SysOp

A SysOp, or System Operator, is a privileged user with access to do anything in
NCA. All installations will require at least one of these just to get set up,
and most likely a dev or system administrator will need this role on an ongoing
basis.

To get the first sysop, follow the installation and configuration instructions,
and then start the server in debug mode:

```bash
./bin/server -c ./settings --debug
```

Debug, among other things, lets you fake a login via
`http://your.site/users?debuguser=<user>`. NCA by default starts with a user
named "sysop" with the sysop privileges. Simply replace `<user>` with `sysop`
and you'll have full control of NCA.

## Create Users

From here you can set up whatever users you need. NCA's role list should
describe the abilities fairly well, and won't be duplicated here.

Users' logins should match whatever authentication system you're using in your
app proxy (e.g., Apache with LDAP). NCA will assume whatever Apache sends in
the `X-Remote-User` header is going to match a user in the local database, and
that the user has in fact been authenticated by Apache.

In security terms: authentication is done by Apache; authorization by NCA.

You will probably want at least one "Site Manager". This person has access to
anything that any other roles have, with the exception of sysops. Site managers
can generally be non-technical people who need to be able to manage the vast
majority of day-to-day NCA processes, such as creating titles, adding and
deactivating users, etc.

## Finalize

Once you have Apache set up to do the authentication, and you have the sysop(s)
and site manager(s) set up, stop NCA's HTTP listener. Subsequent startups
should never use the `--debug` flag in a production environment.

## Development

For development use, `scripts/localdev.sh` already runs the server in debug
mode. Assuming you follow the [development guide's][dev-guide] instructions,
you'll always be in debug mode and able to fake any login, including sysop.

[dev-guide]: <{{% ref "/contributing/dev-guide" %}}>
