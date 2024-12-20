#!/usr/bin/env bash
set -eu

# This script's purpose is to just run through an end-to-end test starting from
# nothing, faking curation, then generating and approving batches. It's a good
# starting point for building other tests, and an okay test when doing a refactor
# that may cause unexpected changes.

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
  wait_db
  queue_batches

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
