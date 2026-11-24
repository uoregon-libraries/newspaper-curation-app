#!/usr/bin/env bash
#
# End to end, from nothing: process uploads, fake curation and review, then
# build and approve batches. A good baseline for refactors, and a starting
# point for new recipes.
#
#   test/recipes/general.sh [name]
#
# See README.md for naming runs and comparing their reports.

source "$(dirname "$0")/../lib.sh"

stage "Reset the stack and load the test issues" prep_stage
stage "Process uploads, then fake page review" process_stage
stage "Fake curation and metadata review" curate_stage
stage "Build batches and load them into staging" batch_stage
stage "Load the approved batches into production" go_live_stage
run_recipe "$@"
