---
title: Testing
weight: 40
description: Automated and manual testing of NCA
---

## Unit Testing

Running unit tests is easy:

    make test

This compiles all of the code and tests any `*_test.go` files.  Test coverage
is spotty at best right now, but the compile-time checks catch the most common
problems, like typos in variable names.

Contributors: feel free to add more unit tests to improve overall coverage!

## Manual Testing

Manually testing NCA can be time-consuming, as you have to find, copy, and then
load issues into NCA, enter metadata, etc. If you repeat this often enough, you
may find the project's "test" directory helpful. It contains documentation and
scripts which are meant to make real-world-like testing a lot easier.

To use the manual test scripts/recipes, you will need to use the recommended
development approach (["Hybrid Developer"][1]) or the scripts' behavior will
not be defined.

View the README.md file in the test directory for details.

[1]: <{{% ref "dev-guide#hybrid-developer" %}}>

### Saving State

At any time you can save and restore the application's state via the top-level
`manage` script.  This script has a variety of commands, but `./manage backup`
and `./manage restore` will back up or restore **all files** in the fake mount
as well as all data volumes for NCA, assuming your docker install puts data in
`/var/lib/docker/volumes` and you don't change the project name from the
default of "nca".

This can be very handy for verifying a process such as generating batches,
where you may want to have the same initial state, but see what happens as you
tweak settings (or code).
