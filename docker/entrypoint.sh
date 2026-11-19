#!/usr/bin/env bash
#
# Entrypoint for the NCA image: web, workers, and cli all use it.
#
# - With the source tree mounted (the "dev" target), NCA binaries are rebuilt
#   before they run, so restarting a container picks up code changes.
# - "server" and "run-jobs" wait for the database and migrate it first.
# - "migrate" does only that: wait for the database, migrate, and exit.
# - Anything else runs as-is.
#
# Migrations hold a lock in the "locks" volume, which every NCA service
# mounts, so containers starting at the same time take turns.
set -euo pipefail

src=/usr/local/nca

# dev_build rebuilds the given NCA binaries if this is the dev image with the
# source mounted. Anything that isn't an NCA binary is skipped.
dev_build() {
  if [[ ! -f $src/go.mod ]] || ! command -v go >/dev/null; then
    return 0
  fi

  local bin targets=()
  for bin in "$@"; do
    if [[ -n $bin && -d $src/src/cmd/$bin ]]; then
      targets+=("bin/$bin")
    fi
  done

  if [[ ${#targets[@]} -gt 0 ]]; then
    echo "Source tree detected; building ${targets[*]}"
    make -C "$src" --no-print-directory "${targets[@]}"
  fi
}

migrate() {
  wait-for-db
  echo "Running database migrations"
  flock /var/local/nca/locks/migrations migrate-database up
}

cmd=${1:-}
case $cmd in
server | run-jobs)
  dev_build "$cmd" migrate-database
  migrate
  ;;
migrate)
  dev_build migrate-database
  migrate
  exit 0
  ;;
*)
  dev_build "$cmd"
  ;;
esac

exec "$@"
