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
stack][1], run by `bin/ncatest`, that's configured entirely by the repository.
Nobody's `.env` or compose override changes how it behaves, and testing never
touches your dev stack's data. `bin/ncatest` loads test issues, stands in for
curators and reviewers, and writes reports you can compare between runs. It
also runs recipes: scripted end-to-end tests that pause when they need you.

View the README.md file in the test directory for details.

[1]: <{{% ref "dev-guide" %}}>

### Saving State

At any time you can save and restore a stack's state: `bin/ncatest save
<name>` and `bin/ncatest restore <name>` save or restore **all files** in the
test stack's workflow tree (`test/fakemount`) as well as its data volumes
(databases, Solr indexes, etc.), using `podman volume export` and `import`.
Go's caches, NCA's issue cache, and the dev image's binaries aren't included,
since they're not application state. `bin/ncadev save` and `restore` do the
same for the dev stack.

This can be very handy for verifying a process such as generating batches,
where you may want to have the same initial state, but see what happens as you
tweak settings (or code).
