#!/usr/bin/env bash
wait_for_database() {
  MAX_TRIES=30
  TRIES=0
  while true; do
    mysqladmin status -unca -h127.0.0.1 -pnca
    st=$?
    if [[ $st == 0 ]]; then
      return 0
    fi

    let TRIES++
    if [[ $TRIES == $MAX_TRIES ]]; then
      echo "ERROR: Unable to connect to the database after $MAX_TRIES attempts"
      return 2
    fi

    sleep 5
  done
}

# Resets the database, deleting and rebuilding all the seed data
resetdb() {
  docker compose down -v
  docker compose up -d db iiif sftpgo
  wait_for_database && migrate && load_seed_data
}

_get_bulk_lccns() {
  ./bin/bulk-issue-queue -c ./settings --type=$1 | awk '{print $1}' | grep -v "^\(Valid\|---\)"
}

# Runs the database reset, then resets test issues in our fake mount area
prep_for_testing() {
  sudo echo -n
  resetdb
  resetfakemount
  bulk_queue_borndigital

  # Reset the IIIF and SFTPGo services since they'll be looking at a mount
  # point we just deleted
  docker compose stop iiif sftpgo
  docker compose rm -f iiif sftpgo
  docker compose up -d iiif sftpgo
}

# Sets up all fake uploads from the test/fakemount dir, then bulk-queues all
# issues without errors
resetfakemount() {
  pushd .
  cd ./test
  ./makemine.sh
  rm fakemount/* -rf
  go run copy-sources.go .
  ./make-older.sh
  popd
}

bulk_queue_borndigital() {
  # Bulk-queue all issues we can
  make bin/bulk-issue-queue || return 1
  pushd .
  cd ./test
  ./makemine.sh
  ./make-older.sh 30
  popd
  for lccn in $(_get_bulk_lccns borndigital); do
    ./bin/bulk-issue-queue -c ./settings --type=borndigital --key=$lccn
  done
}

migrate() {
  goose -dir ./db/migrations/ mysql 'nca:nca@tcp(localhost:3306)/nca' up
}

load_seed_data() {
  mysql -unca -pnca -Dnca -h127.0.0.1 < ./docker/mysql/nca-seed-data.sql
}

upload_server() {
  make bin/upload-server || return 1
  export NCA_DBCONNECT="nca:nca@tcp(localhost:3306)/nca"
  export NCA_SECRET="shhhh"
  ./bin/upload-server --debug --bind-address ":8080" --webroot "http://localhost:8080"
}

server() {
  docker compose up -d db iiif sftpgo
  wait_for_database
  make bin/server || return 1
  echo
  echo "Make sure RAIS knows its URL since it's running in a container:"
  echo "- Update docker-compose.override.yml so RAIS exposes its port"
  echo "- Set RAIS_IIIFBASEURL in .env or the docker override, e.g., RAIS_IIIFBASEURL=http://localhost:12415"
  echo "- Set IIIF_BASE_URL in the NCA settings file, e.g., IIIF_BASE_URL=http://localhost:12415/images/iiif"
  echo "- Restart the iiif service if you changed any of the above since it started last"
  echo
  ./bin/server -c ./settings --debug
}

workers() {
  docker compose up -d db iiif sftpgo
  wait_for_database
  make bin/run-jobs || return 1
  ./bin/run-jobs -c ./settings -v watchall
}
