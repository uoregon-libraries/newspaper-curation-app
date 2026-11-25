# Helpers for NCA's test stack, used by the recipes (test/recipes/*.sh) and
# handy in your own shell:
#
#   source test/lib.sh
#
# - cli <command...>: run a command in a throwaway NCA container, e.g.,
#   "cli queue-batches" or "cli bash"
# - harness <command...>: run a test harness command (see test/harness), e.g.,
#   "harness curate"
# - sql [mariadb args...]: a MariaDB console on NCA's database, or e.g.
#   sql -e "SELECT ..."
# - workonce: stop the workers service, and run jobs in the foreground until
#   there are none left
# - up_for_recipe: start every service but workers, so jobs only run when
#   you run workonce
# - reset_stack [-y] [--issues]: scripts/reset, on the test stack
# - savepoint save|restore <name>: scripts/savepoint, on the test stack
#
# Each prints the command it runs, so you can see what it does (and run it
# yourself). For anything else, run compose on the test stack with
# test/stack/compose.

_nca_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# _show prints a command, quoting what needs it, then runs it from the
# checkout root, so what it prints works from there
_show() {
  local arg words=()
  for arg in "$@"; do
    if [[ $arg =~ ^[A-Za-z0-9_./:=,@%+-]+$ ]]; then
      words+=("$arg")
    else
      words+=("${arg@Q}")
    fi
  done
  printf '+ %s\n' "${words[*]}" >&2
  (cd "$_nca_root" && "$@")
}

# _tty_flag prints -T (don't allocate a TTY) unless we're in a terminal, so
# piping and logging work
_tty_flag() {
  if [[ ! -t 0 || ! -t 1 ]]; then
    echo -T
  fi
}

cli() {
  # shellcheck disable=SC2046 # _tty_flag prints one word or nothing
  _show test/stack/compose --profile tools run --rm --no-deps $(_tty_flag) cli "$@"
}

harness() {
  cli go run ./test/harness "$@"
}

sql() {
  # shellcheck disable=SC2016,SC2046 # the variables are the db container's
  _show test/stack/compose exec $(_tty_flag) db \
    sh -c 'MYSQL_PWD=$MARIADB_PASSWORD exec mariadb -u"$MARIADB_USER" "$MARIADB_DATABASE" "$@"' mariadb "$@"
}

workonce() {
  _show test/stack/compose stop workers &&
    cli run-jobs -v --exit-when-done watchall
}

reset_stack() {
  _show env NCA_COMPOSE=test/stack/compose NCA_PROJECT=nca-test scripts/reset "$@"
}

savepoint() {
  _show env NCA_COMPOSE=test/stack/compose NCA_PROJECT=nca-test NCA_SAVEPOINTS=test/var/savepoints \
    scripts/savepoint "$@"
}

# test_url prints the URL for a port setting (e.g., ONI_STAGING_PORT) in the
# test stack's env file, which test/stack/compose writes every time it runs
test_url() {
  local envfile=$_nca_root/test/var/stack.env
  local host port
  host=$(sed -n 's/^PUBLIC_HOST=//p' "$envfile" | tail -n 1)
  port=$(sed -n "s/^$1=//p" "$envfile" | tail -n 1)
  echo "http://$host:$port"
}

#####
# The recipe runner. A recipe is a script that sources this file, adds its
# stages with "stage", then calls "run_recipe" (see test/recipes/general.sh).
#####

_stage_names=()
_stage_funcs=()

# stage adds a stage to the recipe: a description, and a function that runs it
stage() {
  _stage_names+=("$1")
  _stage_funcs+=("$2")
}

# pause waits for the tester to press Enter
pause() {
  printf '\n%s\n\nPress Enter to continue: ' "$1"
  if ! read -r; then
    echo "No input: recipes that pause need a terminal" >&2
    return 1
  fi
}

# up_for_recipe starts every service but workers. Recipes run jobs with
# workonce, at set points; a workers service starting on its own would race
# them, making runs less repeatable.
up_for_recipe() {
  local services
  mapfile -t services < <(cd "$_nca_root" && test/stack/compose config --services | grep -vx workers)
  if [[ ${#services[@]} -eq 0 ]]; then
    echo "Couldn't list the test stack's services" >&2
    return 1
  fi
  _show test/stack/compose up -d "${services[@]}"
  # Create the workers container without starting it, so "stop" and "down"
  # find one instead of complaining
  _show test/stack/compose up --no-start workers
}

# run_recipe runs the recipe's stages, then writes a report (see the
# harness's "report" command), naming everything after the run: its first
# argument, or "git describe --tags". After each stage, it makes a save point
# named for the stage and run (e.g., "02-baseline"). Running the recipe again
# with the same name restores the last save point and picks up after it.
run_recipe() {
  set -euo pipefail

  if [[ $# -gt 1 ]]; then
    echo "Usage: $0 [name]" >&2
    return 2
  fi
  local run=${1:-}
  if [[ -z $run ]]; then
    if ! run=$(git -C "$_nca_root" describe --tags); then
      echo 'Naming the run with "git describe --tags" failed: give it a name' >&2
      return 1
    fi
  fi

  local report=$_nca_root/test/$run-report
  if [[ -e $report ]]; then
    echo "$report already exists: remove or rename it, or name this run something else" >&2
    return 1
  fi

  # Everything goes to the terminal and a log, so the run can be checked
  # afterward
  local log=$_nca_root/test/var/logs/recipe-$run.log
  mkdir -p "$(dirname "$log")"
  exec > >(tee -a "$log") 2>&1
  echo "Running recipe $(basename "$0" .sh) as \"$run\" (log: $log)"

  _show test/stack/compose --profile tools build

  # Pick up after the last stage that has a complete save point (see
  # scripts/savepoint)
  local i sp start=0
  for ((i = ${#_stage_funcs[@]} - 1; i >= 0; i--)); do
    sp=$(printf '%02d-%s' "$i" "$run")
    if [[ -f $_nca_root/test/var/savepoints/$sp/manifest ]]; then
      printf '\nFound save point %s; restoring it to skip ahead\n' "$sp"
      savepoint restore "$sp"
      start=$((i + 1))
      break
    elif [[ -d $_nca_root/test/var/savepoints/$sp ]]; then
      printf '\nSkipping save point %s: it has no manifest, so its save was interrupted\n' "$sp"
    fi
  done

  for ((i = start; i < ${#_stage_funcs[@]}; i++)); do
    printf '\n=== Stage %02d: %s ===\n\n' "$i" "${_stage_names[i]}"
    if ((i > 0)); then
      up_for_recipe
      cli wait-for-db
    fi
    "${_stage_funcs[i]}"
    savepoint save "$(printf '%02d-%s' "$i" "$run")"
  done

  up_for_recipe
  cli wait-for-db
  harness report --name "$run"

  printf '\nDone. The report is in %s\n' "$report"
  echo 'The workers service is stopped; "test/stack/compose up -d" starts it.'
}

#####
# Stages the recipes share. Every recipe is a variation on the general test.
#####

prep_stage() {
  reset_stack -y --issues
}

process_stage() {
  workonce
  harness rename-page-review
  harness make-older --days 4
  # The job runner rescans the filesystem now and then; restarting it gets
  # the renamed issues moving right away
  workonce
}

curate_stage() {
  harness curate
  workonce
  harness review
  workonce
}

verify_staging() {
  pause "Check that the batches are on staging ONI ($(test_url ONI_STAGING_PORT)), then approve them in NCA ($(test_url PUBLIC_PORT)/nca?debuguser=sysop)."
}

batch_stage() {
  cli queue-batches
  workonce
  verify_staging
}

go_live_stage() {
  workonce
  pause "Check that the batches are on production ONI ($(test_url ONI_PROD_PORT)), and are \"live\" in NCA."
}
