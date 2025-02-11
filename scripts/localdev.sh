#!/usr/bin/env bash
ONI_SERVICES="oni-agent-prod oni-agent-staging oni-prod oni-staging"

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

# Starts the dependent services needed by NCA
start_docker_services() {
  docker compose up -d db iiif sftpgo $ONI_SERVICES
}

# Resets the database, deleting and rebuilding all the seed data
resetdb() {
  docker compose down -v
  start_docker_services
  wait_for_database && migrate && load_seed_data && create_test_users
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

  # Reset the services which share files, since they'll be looking at a mount
  # point we just deleted
  docker compose stop iiif sftpgo $ONI_SERVICES
  docker compose rm -f iiif sftpgo $ONI_SERVICES
  docker compose up -d iiif sftpgo $ONI_SERVICES
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
  make bin/migrate-database
  ./bin/migrate-database -c ./settings up
}

# This is primarily for making the dev setup a lot faster by skipping migrations
reset_and_build_init_sql() {
  echo > docker/init.sql
  docker compose down -v
  dc up -d db
  wait_for_database
  migrate
  source ./settings && mysqldump --host="$DB_HOST" --user="$DB_USER" --password="$DB_PASSWORD" --databases "$DB_DATABASE" --routines --triggers > docker/init.sql
}

load_seed_data() {
  mysql -unca -pnca -Dnca -h127.0.0.1 -e "INSERT INTO mocs (code, name) VALUES
    ('oru','University of Oregon Libraries; Eugene, OR'),
    ('hoodriverlibrary','Hood River County Library District; Hood River, OR');"
  pushd .

  # I hate this hack, but it seems loading titles into ONI interacts with Solr
  # in some disastrous way if ONI hasn't finished setting up the Solr schema
  # bits, and currently we don't have a way to do a simple "are you
  # initialized?" query for ONI
  echo -n "Waiting for ONI staging"
  while true; do
    echo -n "."
    docker compose exec oni-staging curl -s --head localhost >/dev/null && break
    sleep 1
  done
  echo
  echo -n "Waiting for ONI production"
  while true; do
    echo -n "."
    docker compose exec oni-prod curl -s --head localhost >/dev/null && break
    sleep 1
  done
  echo

  cd ./test
  go run load-marc.go -c ../settings
  popd
}

create_test_users() {
  pushd .
  cd ./test
  go run create-test-users.go -c ../settings
  popd
}

upload_server() {
  make bin/upload-server || return 1
  export NCA_DBCONNECT="nca:nca@tcp(localhost:3306)/nca"
  export NCA_SECRET="shhhh"
  ./bin/upload-server --debug --bind-address ":8080" --webroot "http://localhost:8080"
}

server() {
  start_docker_services
  wait_for_database
  SETTINGS_PATH=$(pwd)/settings ./sftpgo/get_admin_api_key.sh --force >/dev/null
  make bin/server || return 1
  echo
  echo "Make sure RAIS knows its URL since it's running in a container:"
  echo "- Update compose.override.yml so RAIS exposes its port"
  echo "- Set RAIS_IIIFBASEURL in .env or the docker override, e.g., RAIS_IIIFBASEURL=http://localhost:12415"
  echo "- Set IIIF_BASE_URL in the NCA settings file, e.g., IIIF_BASE_URL=http://localhost:12415/images/iiif"
  echo "- Restart the iiif service if you changed any of the above since it started last"
  echo
  ./bin/server -c ./settings --debug
}

workers() {
  start_docker_services
  wait_for_database
  make bin/run-jobs || return 1
  ./bin/run-jobs -c ./settings -v watchall
}

workonce() {
  start_docker_services
  wait_for_database
  make bin/run-jobs || return 1
  ./bin/run-jobs -c ./settings -v --exit-when-done watchall
}
