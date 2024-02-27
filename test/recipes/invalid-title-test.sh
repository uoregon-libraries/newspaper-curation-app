#!/usr/bin/env bash
set -eu

# This script is nearly a mirror of the general test, but its purpose is to
# verify that titles with an unvalidated LCCN are properly kept out of batches
# when they're queued up

source test/recipes/testlib.sh
source scripts/localdev.sh

name=$(get_testname)
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
  workonce 2>&1 | tee -a workers.log

  # Stop workers, save state again
  ./manage backup 02-$name
else
  echo "Detected backup 02; skipping processing"
  echo "Restoring backup 02 to begin step 03"
  ./manage restore 02-$name
fi

wait_db

# Hack up one of the titles to say it isn't validated
echo "Setting a title to be invalid"
mysql -unca -pnca -Dnca -h127.0.0.1 -e "
  UPDATE titles
    SET valid_lccn = 0
    WHERE lccn IN (
      SELECT lccn FROM issues WHERE workflow_step = 'ReadyForBatching'
      ORDER BY lccn
    )
    LIMIT 1
"

# Generate batches
./bin/queue-batches -c ./settings

# Start workers, wait for jobs to complete (~30sec)
workonce 2>&1 | tee -a workers.log

echo "Approve batches manually in NCA, then press [ENTER] continue"
read
workonce 2>&1 | tee -a workers.log

# Batches' statuses in NCA should read "passed_qc", no jobs should be anything
# other than "success"

echo "Verify all batches' statuses are 'passed_qc', then press [ENTER] to continue"
read

cd test
./report.sh $name
cd ..
echo "DONE"
