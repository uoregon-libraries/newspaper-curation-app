#!/usr/bin/env bash
#
# Dev helpers for NCA's container stack. Source this from the NCA root:
#
#   source scripts/localdev.sh
#
# Everything runs in containers. You need `.env` and `compose.override.yml`
# set up first; see the Developer's Guide (hugo/content/contributing/dev-guide.md).
#
# Day to day:
#
# - nca_up: start (or update) the stack
# - nca_compose ...: compose with your profiles, e.g., `nca_compose restart web`
# - ncacli <command> [args]: run anything in a throwaway NCA container, e.g.,
#   `ncacli queue-batches`. No "-c" needed. In dev, NCA binaries are rebuilt
#   from your checkout first.
# - ncatest <command> [args]: ncacli, but from the test/ directory, e.g.,
#   `ncatest go run run-workflow.go --operation curate`
# - ncasql [args]: a mysql client on NCA's database
# - resetdb: wipe the databases (NCA's and both ONIs') and reseed them
# - prep_for_testing: resetdb, then reload test issues into test/fakemount

NCA_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# Compose, with COMPOSE_PROFILES honored on old podman-compose versions
nca_compose() {
  "$NCA_ROOT/scripts/compose" "$@"
}

# The compose project name, which prefixes every volume. Same precedence as
# compose: the environment, then .env, then the checkout's directory name.
_nca_project() {
  local name=${COMPOSE_PROJECT_NAME:-}
  if [[ -z $name && -f $NCA_ROOT/.env ]]; then
    name=$(sed -n 's/^[[:space:]]*COMPOSE_PROJECT_NAME[[:space:]]*=//p' "$NCA_ROOT/.env" |
      tail -n 1 | sed 's/[[:space:]]#.*//' | tr -d "\"'[:space:]")
  fi
  echo "${name:-$(basename "$NCA_ROOT")}"
}

# Removes the given volumes (by their compose name, e.g., "db") if they exist
_nca_rm_volumes() {
  local project vol
  project=$(_nca_project)
  for vol in "$@"; do
    if podman volume exists "${project}_$vol"; then
      podman volume rm "${project}_$vol" >/dev/null || return 1
    fi
  done
}

# Use -T (no TTY) unless we're attached to a terminal, so piping and
# redirection work
_nca_tflag() {
  if [[ ! -t 0 || ! -t 1 ]]; then
    echo "-T"
  fi
}

ncacli() {
  # shellcheck disable=SC2046
  nca_compose --profile tools run --rm --no-deps $(_nca_tflag) cli "$@"
}

ncatest() {
  # shellcheck disable=SC2046
  nca_compose --profile tools run --rm --no-deps $(_nca_tflag) -w /usr/local/nca/test cli "$@"
}

# shellcheck disable=SC2016
ncasql() {
  ncacli bash -c 'MYSQL_PWD=$NCA_DB_PASSWORD exec mysql -h"$NCA_DB_HOST" -u"$NCA_DB_USER" "$NCA_DB_DATABASE" "$@"' mysql "$@"
}

# Creates the host directories the dev stack needs before anything can start:
# the override binds volumes to test/fakemount and tmp/issue-cache, and
# CONTAINER_LOG_DIR has to exist
_nca_host_dirs() {
  local dir
  for dir in backup/originals batch-archive errors outgoing page-review production-batches scans sftp workflow; do
    mkdir -p "$NCA_ROOT/test/fakemount/$dir" || return 1
  done
  mkdir -p "$NCA_ROOT/tmp/issue-cache" "$NCA_ROOT/logs"
}

nca_up() {
  _nca_host_dirs && nca_compose up -d "$@"
}

# Starts everything but NCA itself (web, workers, and the proxy in front of
# them), for setup that mustn't race the job runner
_nca_up_deps() {
  local services
  services=$(nca_compose config --services 2>/dev/null | grep -v -x -E 'web|workers|proxy|cli') || return 1
  # shellcheck disable=SC2086
  nca_up $services
}

wait_for_database() {
  ncacli wait-for-db
}

# Runs migrations, waiting for the database if necessary. Safe to run while
# web and workers are starting: they all share a lock.
migrate() {
  ncacli migrate
}

# Wipes NCA's database and both ONIs' databases and Solr indexes, then
# reseeds NCA with MARC org codes, titles (also loaded into both ONIs), and
# one test user per role. Go's caches and NCA's issue cache are kept.
resetdb() {
  _resetdb && nca_up
}

# resetdb, but leaves NCA itself stopped
_resetdb() {
  nca_compose down || return 1
  _nca_rm_volumes db locks sftpgo-home sftpgo-backups \
    oni-staging-data-mariadb oni-staging-data-solr \
    oni-prod-data-mariadb oni-prod-data-solr || return 1
  _nca_up_deps || return 1
  migrate && load_seed_data && create_test_users
}

# Only keeps what looks like an LCCN: the first run of "cli" builds its image,
# and podman-compose prints the image ID with the command's output
_get_bulk_lccns() {
  ncacli bulk-issue-queue --type="$1" | awk '{print $1}' | grep -E '^[a-z]{0,3}[0-9]{8,10}$'
}

# Runs the database reset, then resets test issues in our fake mount area.
# NCA stays stopped until the end so the job runner can't start on issues
# while the fake mount is being replaced.
prep_for_testing() {
  _resetdb || return 1
  resetfakemount || return 1
  bulk_queue_borndigital || return 1

  # Every service touching the fake mount needs to see a consistent view, and
  # it's painful enough when something's stale that we just take everything
  # down and back up again.
  nca_compose down && nca_up
}

# Empties the fake mount, then copies all test sources into it (see
# test/README.md). copy-sources.go runs in a container, so it can delete
# files containers created, whoever owns them on the host.
resetfakemount() {
  _nca_host_dirs || return 1
  ncatest go run copy-sources.go . || return 1
  (cd "$NCA_ROOT/test" && ./make-older.sh)
}

bulk_queue_borndigital() {
  (cd "$NCA_ROOT/test" && ./make-older.sh 30) || return 1
  local lccn
  for lccn in $(_get_bulk_lccns borndigital); do
    ncacli bulk-issue-queue --type=borndigital --key="$lccn" || return 1
  done
}

# Regenerates docker/init.sql, the migrated-schema snapshot new databases start
# from, then rebuilds the db image with it. This wipes NCA's database.
reset_and_build_init_sql() {
  nca_compose down || return 1
  _nca_rm_volumes db locks || return 1
  _nca_host_dirs || return 1
  nca_compose up -d db || return 1
  migrate || return 1
  # shellcheck disable=SC2016
  nca_compose exec -T db sh -c 'MYSQL_PWD=$MARIADB_PASSWORD exec mariadb-dump -u"$MARIADB_USER" \
    --databases "$MARIADB_DATABASE" --routines --triggers' >"$NCA_ROOT/docker/init.sql" || return 1
  nca_compose build db
}

load_seed_data() {
  ncasql -e "INSERT INTO mocs (code, name) VALUES
    ('oru','University of Oregon Libraries; Eugene, OR'),
    ('hoodriverlibrary','Hood River County Library District; Hood River, OR');" || return 1

  # Loading titles into ONI interacts with Solr in some disastrous way if ONI
  # hasn't finished setting up Solr, and ONI has no simple "are you
  # initialized?" query
  local env
  for env in staging prod; do
    echo -n "Waiting for ONI $env"
    until nca_compose exec -T "oni-$env" curl -s --head localhost >/dev/null 2>&1; do
      echo -n "."
      sleep 1
    done
    echo
  done

  ncatest go run load-marc.go
}

create_test_users() {
  ncatest go run create-test-users.go
}

# Runs jobs until there are none left, verbosely, in place of the workers
# service (which is stopped so they don't fight over jobs). Start it again
# with `nca_compose start workers`.
workonce() {
  nca_compose stop workers || return 1
  ncacli run-jobs -v --exit-when-done watchall
}
