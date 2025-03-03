---
title: Add Privileges / Roles
weight: 30
description: Adding a new privilege and making it do something
---

New privileges require a lot of different changes in order to create them, tie
them to a role, and then have NCA use them.

- Edit `src/privilege/role.go` if the new privilege(s) are going to be tied to
  an entirely new role.
- Edit `src/privilege/privilege.go` and add the item in the big list of vars.
  You have to define what role(s) can have said privilege.
  - SysOps always have all privileges and aren't specified explicitly, but the
    privilege does require a definition, even if it's just an empty role list.
  - Examples:
    - `PrivA = newPrivilege(RoleCurator)`: `PrivA` is explicitly given to
      curators and implicitly to sysops. No other users will have PrivA access.
    - `PrivB = newPrivilege()`: `PrivB` is implicitly given to sysops. Nobody
      else will have this privilege.
- If the privilege needs to be used in handlers, use a middleware function.
  - The `audithandler` has a simple example of this, where a `canView` function
    wraps access to all routes.
  - Fairly complex authorization middleware functions can be seen in the
    `workflowhandler` code, where the authorization functions verify not just
    privileges, but also issue state, issue ownership, etc.
  - In almost all situations where a new route or handler is created, an access
    check of *some* kind should be created.
- If the privilege needs a check in the HTML templates:
  - First, you have to expose the privilege by name in
    `src/cmd/server/internal/responder/templates.go`. There's a long list of
    privileges there, exposed as functions, to help ensure compile-time
    correctness of privilege checks.
  - Second, you have to use the privilege. `templates/layout.go.html` has
    examples of using `.User.PermittedTo` for deciding which navigation items
    to expose.
- Occasionally you may think it necessary to add a manual permissions check
  somewhere in a handler. Usually this is a bad idea, but if you're certain
  such a check is necessary, you can find a few examples of this in various
  handlers. If you don't want to spend the time to figure out where they are
  and how to emulate them, you probably don't need them that badly :P
