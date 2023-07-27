#!/usr/bin/env bash
set -eu

# This script's purpose is to just run through an end-to-end test starting from
# nothing, faking curation, then generating and approving batches. It's a good
# starting point for building other tests, and an okay test when doing a refactor
# that may cause unexpected changes.

wait_db() {
  set +e && wait_for_database && set -e
}

name=${1:-}
if [[ $name == "" ]]; then
  name=$(git describe --tags)
  echo "No name was provided; using commit information from git: name is '$name'"
fi

make clean
make

source scripts/localdev.sh

if [[ ! -d ./backup/00-$name ]]; then
  prep_for_testing

  # Save state using the "name" variable from above
  ./manage backup 00-$name
else
  echo "Detected backup 00; skipping processing"
  if [[ ! -d ./backup/01-$name ]]; then
    echo "Restoring backup 00 to begin step 01"
    ./manage restore 00-$name
  fi
fi

if [[ ! -d ./backup/01-$name ]]; then
  wait_db

  # Wait until DB is up and start workers
  ./bin/run-jobs -c ./settings watchall --exit-when-done

  # Wait for jobs to finish

  # Renumber page review PDFs and hack their date
  cd test
  ./rename-page-review.sh
  ./make-older.sh
  cd ..

  # If necessary, restart workers to make the PDF mover re-read the filesystem quicker
  ./bin/run-jobs -c ./settings watchall --exit-when-done

  # Wait for jobs to complete (~10min)

  # Stop workers, save state
  ./manage backup 01-$name
else
  echo "Detected backup 01; skipping processing"
  if [[ ! -d ./backup/02-$name ]]; then
    echo "Restoring backup 01 to begin step 02"
    ./manage restore 01-$name
  fi
fi

if [[ ! -d ./backup/02-$name ]]; then
  wait_db

  # Fake-curate, fake-review
  cd test
  go run run-workflow.go -c ../settings --operation curate
  go run run-workflow.go -c ../settings --operation review
  cd ..

  # Start workers, wait for jobs to complete (~30sec)
  ./bin/run-jobs -c ./settings watchall --exit-when-done

  # Stop workers, save state again
  ./manage backup 02-$name
else
  echo "Detected backup 02; skipping processing"
  echo "Restoring backup 02 to begin step 03"
  ./manage restore 02-$name
fi

wait_db

# Generate batches
./bin/queue-batches -c ./settings

# Start workers, wait for jobs to complete (~30sec)
./bin/run-jobs -c ./settings watchall --exit-when-done

echo "Approve batches manually in NCA, then press [ENTER] continue"
read
./bin/run-jobs -c ./settings watchall --exit-when-done

# Batches' statuses in NCA should read "passed_qc", no jobs should be anything
# other than "success"

echo "Verify all batches' statuses are 'passed_qc', then press [ENTER] to continue"
read

cd test
./report.sh $name
cd ..
echo "DONE"
