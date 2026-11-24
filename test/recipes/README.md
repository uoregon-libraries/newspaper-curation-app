# Test Recipes

This directory contains a handful of (hopefully) helpful testing recipes,
using NCA's test stack (see the [test README](../README.md)). Each file
contains an explanation and some general information about what's being
tested and how it should be used.

At a high level, these test recipes are meant to be used to make sure that a
new function or refactor doesn't change anything you don't expect it to change,
so you're generally choosing some kind of baseline to test against, not just
running the recipe in isolation.

Some general / common information follows.

## Scripted Recipes

Some recipes are scripts, built on the test stack's helpers in `test/lib.sh`:

- `general.sh`: end to end, from nothing: process uploads, fake curation and
  review, then build and approve batches. It's a good baseline for refactors,
  and a starting point for new recipes.
- `invalid-title.sh`: the general test, but one title's LCCN is marked invalid
  before batching, to verify that titles with an unvalidated LCCN are kept out
  of batches
- `broken-oni.sh`: the general test, but you stop staging ONI's Solr while
  batches load, to test NCA's handling of external job failures (entwined job
  retry / restart)

Run one with a name for the run, e.g., `test/recipes/general.sh baseline`.
It creates a completely fresh environment, runs each stage, in some cases
directs you to take actions manually in NCA, and eventually creates a report.
You should only have to check out the branches you need and run the recipe for
each branch, then compare reports. It prints every command it runs, so if a
stage goes wrong, you can see what it did and repeat any step by hand.

After each stage, the recipe makes a save point named for the stage and run
(e.g., `02-baseline`). Running the recipe again with the same name restores
the last save point and picks up from there, so an interrupted run doesn't
start over. To repeat a stage, delete its save point (and any later ones) from
`test/var/savepoints`.

Everything the recipe runs is logged to `test/var/logs/recipe-<name>.log`.

A new recipe is a short script: copy `general.sh`, and replace or add stages.
Each stage is a shell function, using the helpers in `test/lib.sh`.

## Step-by-step Recipes

For markdown documents, *don't just copy and paste the shell code!* We're
putting stuff in here as we go, and sometimes correcting our processes after
the fact and trying to keep the recipe in sync. Some file locations may be
environment-specific. Additionally, commands may change or even be removed in
the future. Consider these recipes a guide, not a blind process to run!

## Names

Each recipe should be run once on your baseline (usually the `main` branch) and
again on your fixed code. Snapshots (save points, which back up data state in
case you need to repeat a step) and reports need meaningful names.

Scripted recipes take the name as an argument. With no name, they use `git
describe --tags`. For step-by-step recipes, it's easiest to do something like
`export name=baseline`, then use `$name` when you make save points or
reports, e.g., `harness report --name $name`.

## Repeat And Compare

You'll run the recipe twice: once on your baseline, once on your fix/refactor.
The test stack's containers build NCA from your checkout as they start, so
there's nothing to compile first.

After each run, you should generate a report (scripted recipes do this for
you). Once you have two reports, you can compare them using git. Example:

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

`workonce` (from `test/lib.sh`) runs jobs until there are none left. To check
on jobs by hand:

    sql -e "select * from jobs where status not in ('success', 'on_hold')"

Hopefully we get a UI for viewing jobs one day....
