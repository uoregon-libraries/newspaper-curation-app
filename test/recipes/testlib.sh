#!/usr/bin/env bash
set -eu

wait_db() {
  start_docker_services
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
  make fast
  rm -f workers.log
}

load_titles() {
  cd test
  go run load-marc.go -c ../settings
  cd ..
}

prep_and_backup_00() {
  if [[ ! -d ./backup/00-$name ]]; then
    prep_for_testing
    docker compose up -d oni-agent-staging oni-agent-prod
    sleep 1
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
  ./bin/queue-batches -c ./settings 2>&1 | tee -a queue-batches.log

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
