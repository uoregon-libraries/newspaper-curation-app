---
title: Testing
weight: 40
description: Automated and manual testing of NCA
---

## Unit Testing

Running unit tests is easy:

```bash
make test
```

This compiles all of the code and tests any `*_test.go` files. Test coverage is
spotty at best right now, but the compile-time checks, and static analysis run
by `make`, catch most of the kinds of problems you'd test for in
PHP/Ruby/Python/etc (variable typos, misuse of a function, etc.)

Contributors: feel free to add more unit tests to improve overall coverage!

## Manual Testing

Manually testing NCA can be time-consuming, as you have to find, copy, and then
load issues into NCA, enter metadata, etc. If you repeat this often enough, you
may find the project's "test" directory helpful. It contains documentation and
tools which are meant to make real-world-like testing a lot easier.

Manual tests run on a separate test stack: a second copy of the [dev
stack][1], run with `test/stack/compose`, that's configured entirely by the
repository. Nobody's `.env` or compose override changes how it behaves, and
testing never touches your dev stack's data. The test directory has helpers
that load test issues, stand in for curators and reviewers, and write reports
you can compare between runs, plus recipes: scripted end-to-end tests that
pause when they need you.

View the README.md file in the test directory for details.

[1]: <{{% ref "dev-guide" %}}>

### Saving State

At any time you can save and restore a stack's state with
`scripts/savepoint save <name>` and `scripts/savepoint restore <name>`. A save
point holds **all files** in the stack's workflow tree as well as its data
volumes (databases, Solr indexes, etc.), using `podman volume export` and
`import`. Go's caches, NCA's issue cache, and the dev image's binaries aren't
included, since they're not application state. Dev save points go in
`backup/`; the test stack's helpers (see the test README) keep its save
points in `test/var/savepoints`.

This can be very handy for verifying a process such as generating batches,
where you may want to have the same initial state, but see what happens as you
tweak settings (or code).
