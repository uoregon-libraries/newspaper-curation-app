#!/usr/bin/env bash
set -eu

# This script's purpose is to just run through an end-to-end test, but guiding
# the tester to deliberately break ONI to test NCA's handling of external job
# failures (entwined job retry / restart).

source test/recipes/testlib.sh
source scripts/localdev.sh

name=$(get_testname ${1:-})
build_and_clean
prep_and_backup_00

if [[ ! -d ./backup/01-$name ]]; then
  wait_db

  # Wait until DB is up and start workers
  workonce 2>&1 | tee -a workers.log

  # Wait for jobs to finish

  # Renumber page review PDFs and hack their date
  cd test
  ./rename-page-review.sh
  ./make-older.sh
  cd ..

  # If necessary, restart workers to make the PDF mover re-read the filesystem quicker
  workonce 2>&1 | tee -a workers.log

  # Wait for jobs to complete (~10min)

  # Stop workers, save state
  ./manage backup 01-$name
else
  echo "Detected backup 01; skipping processing"
  if [[ ! -d ./backup/02-$name ]]; then
    echo "Restoring backup 01"
    ./manage restore 01-$name
  fi
fi

if [[ ! -d ./backup/02-$name ]]; then
  curate_and_review

  # Stop workers, save state again
  ./manage backup 02-$name
else
  echo "Detected backup 02; skipping processing"
  if [[ ! -d ./backup/03-$name ]]; then
    echo "Restoring backup 02"
    ./manage restore 02-$name
  fi
fi

if [[ ! -d ./backup/03-$name ]]; then
  make bin/run-jobs
  start_docker_services
  wait_db
  ./bin/queue-batches -c ./settings 2>&1 | tee -a queue-batches.log

  echo "Disable Solr in staging, then press [ENTER] continue."
  read

  echo
  echo "Ready to continue. DO NOT re-enable Solr until you've seen enough 'oni_wait_for_job'"
  echo "failures that you're confident in the entwine-retry process."
  echo
  echo "Press [ENTER] to start the job runner"
  read

  # Start workers, wait for jobs to complete (~30sec). Do NOT use the
  # "workonce" shortcut, as it will restart solr!
  ./bin/run-jobs -c ./settings -v --exit-when-done watchall

  echo "Verify batches are on ONI staging, approve them in NCA, then press [ENTER] continue"
  read
  ./manage backup 03-$name
else
  echo "Detected backup 03; skipping processing"
  if [[ ! -d ./backup/04-$name ]]; then
    echo "Restoring backup 03"
    ./manage restore 03-$name
  fi
fi

if [[ ! -d ./backup/04-$name ]]; then
  wait_db
  run_batch_jobs

  ./manage backup 04-$name
else
  echo "Detected backup 04; skipping processing"
  echo "Restoring backup 04"
  ./manage restore 04-$name
fi

wait_db
go run test/report.go -c ./settings --dir=$(pwd)/test --name=$name
echo "DONE"
