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
  echo "You must enter a name for backups and reports"
  exit 1
fi

make clean
make

source scripts/localdev.sh
prep_for_testing

# Save state using the "name" variable from above
./manage backup 00-$name
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
wait_db

# Generate batches
./bin/queue-batches -c ./settings

# Start workers, wait for jobs to complete (~30sec)
./bin/run-jobs -c ./settings watchall --exit-when-done

read -n 1 -s -r -p "Approve batches manually in NCA, then press any key to continue"
echo
./bin/run-jobs -c ./settings watchall --exit-when-done

# Batches' statuses in NCA should read "passed_qc", no jobs should be anything
# other than "success"

read -n 1 -s -r -p "Verify all batches' statuses are 'passed_qc', then press any key to continue"
echo

cd test
./report.sh $name
cd ..
echo "DONE"