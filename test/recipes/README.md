# Test Recipes

This directory contains a handful of (hopefully) helpful testing recipes, using
NCA's manual test workflow. Each document contains an explanation and some
general information about what's being tested and how it should be used.

At a high level, these test recipes are meant to be used to make sure that a
new function or refactor doesn't change anything you don't expect it to change,
so you're generally choosing some kind of baseline to test against, not just
running the recipe in isolation.

Some general / common information follows.

## Disclaimer

*Don't just copy and paste the shell code!* We're putting stuff in here as we
go, and sometimes correcting our processes after the fact and trying to keep
the recipe in sync. Some file locations may be environment-specific.
Additionally, commands may change or even be removed in the future. Consider
these recipes a guide, not a blind process to run!

## Names

Each recipe should be run once on your baseline (usually the `main` branch) and
again on your fixed code. When making snapshots (backing up data state in case
you need to repeat a step) and reports will need a meaningful name.

Generally it's easiest to do something like `export name=baseline`, then use
`$name` when you make backups or reports, e.g., `cd test && ./report.sh $name`.

## Repeat And Compare

Again, you'll run the recipe twice: once on your baseline, once on your
fix/refactor. After each run, you should generate a report. Once you have two
reports, you can compare them using git. Example:

```bash
cp -r ./test/baseline-report ./test/r
git add ./test/r
git commit -m "UNDO"
rm ./test/r -rf
cp -r ./test/fix-report ./test/r
git diff ./test/r
git status ./test/r
```

## Waiting for Jobs

To wait for jobs to complete, at the moment I just check the database manually:

    select * from jobs where status not in ('success', 'on_hold');

Hopefully we get a UI for viewing jobs one day....
