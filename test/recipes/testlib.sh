#!/usr/bin/env bash
set -eu

wait_db() {
  set +e && wait_for_database && set -e
}

get_testname() {
  name=${1:-}
  if [[ $name == "" ]]; then
    name=$(git describe --tags)
    echo "No name was provided; using commit information from git: name is '$name'" >&2
  fi
  echo $name
}

build_and_clean() {
  make clean
  make
  rm -f workers.log
}

prep_and_backup_00() {
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
}

finish_batches() {
  wait_db

  # Generate batches
  ./bin/queue-batches -c ./settings

  # Start workers, wait for jobs to complete (~30sec)
  workonce 2>&1 | tee -a workers.log

  echo "Approve batches manually in NCA, then press [ENTER] continue"
  read
  workonce 2>&1 | tee -a workers.log

  echo "Verify all batches' statuses are 'live', then press [ENTER] to continue"
  read
}

curate_and_review() {
  wait_db

  # Fake-curate, fake-review
  cd test
  go run run-workflow.go -c ../settings --operation curate
  cd ..

  # Start workers, wait for jobs to complete (~30sec)
  workonce 2>&1 | tee -a workers.log

  cd test
  go run run-workflow.go -c ../settings --operation review
  cd ..

  # Start workers, wait for jobs to complete (~30sec)
  workonce 2>&1 | tee -a workers.log
}
