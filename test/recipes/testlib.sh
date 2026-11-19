#!/usr/bin/env bash
set -eu

wait_db() {
  nca_up
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

# Makes sure images are current. NCA's binaries are built from your checkout
# whenever a container starts, so there's nothing to compile here.
build_and_clean() {
  nca_compose build
  nca_compose --profile tools build cli
  rm -f workers.log
}

load_titles() {
  ncatest go run load-marc.go
}

prep_and_backup_00() {
  if [[ ! -d ./backup/00-$name ]]; then
    prep_for_testing
    load_titles

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

queue_batches() {
  wait_db

  # Generate batches
  ncacli queue-batches 2>&1 | tee -a queue-batches.log

  # Start workers, wait for jobs to complete (~30sec)
  workonce 2>&1 | tee -a workers.log

  echo "Verify batches are on ONI staging, approve them in NCA, then press [ENTER] continue"
  read
}

run_batch_jobs() {
  workonce 2>&1 | tee -a workers.log

  echo "Verify batches are in ONI production and are 'live' in NCA, then press [ENTER] to continue"
  read
}

curate_and_review() {
  wait_db

  # Fake-curate, fake-review
  ncatest go run run-workflow.go --operation curate

  # Start workers, wait for jobs to complete (~30sec)
  workonce 2>&1 | tee -a workers.log

  ncatest go run run-workflow.go --operation review

  # Start workers, wait for jobs to complete (~30sec)
  workonce 2>&1 | tee -a workers.log
}

# Writes the end-of-test report to test/<name>-report
make_report() {
  wait_db
  ncatest go run report.go --dir=/usr/local/nca/test --name="$1"
}
