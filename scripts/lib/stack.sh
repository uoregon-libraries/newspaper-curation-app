# Shared by scripts/reset and scripts/savepoint: which stack to work on, and
# where its state is. Source it; don't run it.
#
# Both work on the dev stack, unless these variables say otherwise (the test
# stack's helpers, in test/lib.sh, set them):
#
# - NCA_COMPOSE: the command that runs compose on the stack, split on spaces
#   (default: "podman compose")
# - NCA_PROJECT: the stack's compose project, which prefixes its container
#   and volume names (default: the dev stack's: COMPOSE_PROJECT_NAME from
#   your shell or .env, else the checkout's directory name, as compose does)

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
export PODMAN_COMPOSE_WARNING_LOGS=${PODMAN_COMPOSE_WARNING_LOGS:-false}

read -ra compose <<<"${NCA_COMPOSE:-podman compose}"

project=${NCA_PROJECT:-${COMPOSE_PROJECT_NAME:-}}
if [[ -z $project && -f .env ]]; then
  project=$(sed -n 's/^[[:space:]]*COMPOSE_PROJECT_NAME[[:space:]]*=//p' .env | tail -n 1 |
    sed 's/[[:space:]]#.*//' | tr -d "\"'[:space:]")
fi
project=${project:-$(basename "$PWD")}

die() {
  echo "$*" >&2
  exit 1
}

# Volumes that aren't the stack's state: Go's caches, the dev image's
# compiled binaries, and NCA's issue cache (it can be rebuilt, just slowly)
skip_volumes=" go-cache dev-bin issue-cache "

# declared_volumes prints the names of the volumes the stack's compose files
# declare: the keys of the top-level "volumes" in compose's normalized config
declared_volumes() {
  "${compose[@]}" config | awk '
    /^[^ ]/ { in_volumes = ($0 == "volumes:"); next }
    in_volumes && /^  [^ ]/ { sub(/^  /, ""); sub(/:.*/, ""); print }
  '
}

# scan_volumes sorts the stack's volumes, by their compose names (e.g., "db"):
#
# - data_vols: state volumes podman manages (databases, Solr indexes, etc.)
# - state_binds: state volumes bound to host directories (e.g., the workflow
#   tree), mapped to their directories. A volume whose directory is inside
#   another's (e.g., batches, in the workflow tree) is left out: saving or
#   emptying the outer directory takes care of it.
# - bind_dirs: every directory the stack's volumes are bound to
#
# Only volumes the compose files declare count: a volume left over from an
# older setup may be bound to a directory that isn't this stack's anymore.
# Volumes only exist once compose has created them (e.g., "up").
scan_volumes() {
  data_vols=()
  bind_dirs=()
  declare -gA state_binds=()

  local declared vol name info other
  local -A binds=()
  declared=$(declared_volumes)
  if [[ -z $declared ]]; then
    die "Couldn't read the stack's volumes from its compose config"
  fi
  for name in $declared; do
    vol=${project}_$name
    if ! podman volume exists "$vol"; then
      continue
    fi
    info=$(podman volume inspect --format '{{index .Options "o"}}|{{index .Options "device"}}' "$vol")
    if [[ ${info%%|*} == *bind* ]]; then
      bind_dirs+=("${info#*|}")
      if [[ $skip_volumes != *" $name "* ]]; then
        binds[$name]=${info#*|}
      fi
    elif [[ $skip_volumes != *" $name "* ]]; then
      data_vols+=("$name")
    fi
  done

  for name in "${!binds[@]}"; do
    for other in "${!binds[@]}"; do
      if [[ $other != "$name" && ${binds[$name]} == "${binds[$other]}"/* ]]; then
        continue 2
      fi
    done
    state_binds[$name]=${binds[$name]}
  done
}

# wipe_state stops the stack and deletes its state: its data volumes, and
# everything in its bind-mounted directories. The directories stay, since
# the mounts need them. Containers can write files that only root in
# podman's user namespace can delete, hence "podman unshare".
wipe_state() {
  "${compose[@]}" down
  scan_volumes

  local name dir
  for name in "${data_vols[@]}"; do
    echo "Removing volume $name"
    podman volume rm "${project}_$name" >/dev/null
  done
  for name in "${!state_binds[@]}"; do
    dir=${state_binds[$name]}
    if [[ -d $dir ]]; then
      echo "Emptying $dir"
      podman unshare find "$dir" -mindepth 1 -delete
    fi
  done

  # Emptying a directory deletes any bind-mounted directories inside it
  if [[ ${#bind_dirs[@]} -gt 0 ]]; then
    mkdir -p "${bind_dirs[@]}"
  fi
}
