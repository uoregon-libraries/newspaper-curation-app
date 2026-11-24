#!/usr/bin/env bash
#
# The general test, but one title's LCCN is marked invalid before batching,
# so its issues must be kept out of batches.
#
#   test/recipes/invalid-title.sh [name]
#
# See README.md for naming runs and comparing their reports.

source "$(dirname "$0")/../lib.sh"

invalid_title_batch_stage() {
  echo "Marking one title with issues ready for batching as having an invalid LCCN"
  sql -e "
    UPDATE titles
    SET valid_lccn = 0
    WHERE lccn IN (
      SELECT lccn FROM issues WHERE workflow_step = 'ReadyForBatching'
      ORDER BY lccn
    )
    LIMIT 1"
  batch_stage
}

stage "Reset the stack and load the test issues" prep_stage
stage "Process uploads, then fake page review" process_stage
stage "Fake curation and metadata review" curate_stage
stage "Invalidate a title's LCCN, then build batches and load them into staging" invalid_title_batch_stage
stage "Load the approved batches into production" go_live_stage
run_recipe "$@"
