#!/usr/bin/env bash
#
# The general test, but you stop staging ONI's Solr while batches load, to
# test how NCA handles ONI jobs failing (entwined job retries).
#
#   test/recipes/broken-oni.sh [name]
#
# See README.md for naming runs and comparing their reports.

source "$(dirname "$0")/../lib.sh"

broken_oni_batch_stage() {
  cli queue-batches
  pause "Stop staging ONI's Solr. In another terminal:

    test/stack/compose stop oni-staging-solr"
  pause "Next, the job runner starts, and loading batches into staging fails. Once you've seen enough
\"oni_wait_for_job\" failures to be confident in the retries, start Solr again so the jobs can finish:

    test/stack/compose start oni-staging-solr"
  workonce
  verify_staging
}

stage "Reset the stack and load the test issues" prep_stage
stage "Process uploads, then fake page review" process_stage
stage "Fake curation and metadata review" curate_stage
stage "Build batches, and break staging ONI while they load" broken_oni_batch_stage
stage "Load the approved batches into production" go_live_stage
run_recipe "$@"
