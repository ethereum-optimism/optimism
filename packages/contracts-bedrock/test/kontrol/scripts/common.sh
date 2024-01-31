#!/bin/bash
# Common functions and variables for run-kontrol.sh and make-summary-deployment.sh

notif() { echo "== $0: $*" >&2 ; }
usage() {
  echo "Usage: $0 [-h|--help] [container|local|dev]" 1>&2
  echo "Options:" 1>&2
  echo "  -h, --help         Display this help message." 1>&2
  echo "  container          Run in docker container. Reproduce CI execution. (Default)" 1>&2
  echo "  local              Run locally, enforces registered versions.json version for better reproducibility. (Recommended)" 1>&2
  echo "  dev                Run locally, does NOT enforce registered version. (Useful for developing with new versions and features)" 1>&2
  exit 0
}

# Set Run Directory <root>/packages/contracts-bedrock
WORKSPACE_DIR=$( cd "${SCRIPT_HOME}/../../.." >/dev/null 2>&1 && pwd )

# Variables
export CONTAINER_NAME=kontrol-tests
KONTROLRC=$(jq -r .kontrol < "${WORKSPACE_DIR}/../../versions.json")
export KONTROL_RELEASE=${KONTROLRC}
export LOCAL=false

# Argument Parsing
parse_args() {
  if [ $# -gt 1 ]; then
    usage
  elif [ $# -eq 0 ] || [ "$1" == "container" ]; then
    notif "Running in docker container (DEFAULT)"
    export LOCAL=false
  elif [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
    usage
  elif [ "$1" == "local" ]; then
    notif "Running with LOCAL install, .kontrolrc CI version ENFORCED"
    check_kontrol_version
  elif [ "$1" == "dev" ]; then
    notif "Running with LOCAL install, IGNORING .kontrolrc version"
    export LOCAL=true
    pushd "${WORKSPACE_DIR}" > /dev/null || exit
  else
    usage
  fi
}

check_kontrol_version() {
  if [ "$(kontrol version | awk -F': ' '{print$2}')" == "${KONTROLRC}" ]; then
    notif "Kontrol version matches ${KONTROLRC}"
    export LOCAL=true
    pushd "${WORKSPACE_DIR}" > /dev/null || exit
  else
    notif "Kontrol version does NOT match ${KONTROLRC}"
    notif "Please run 'kup install kontrol --version v${KONTROLRC}'"
    exit 1
  fi
}

start_docker () {
  docker run                                    \
    --name "${CONTAINER_NAME}"                  \
    --rm                                        \
    --interactive                               \
    --detach                                    \
    --env FOUNDRY_PROFILE="${FOUNDRY_PROFILE}"  \
    --workdir /home/user/workspace              \
    runtimeverificationinc/kontrol:ubuntu-jammy-"${KONTROL_RELEASE}"

  # Copy test content to container
  docker cp --follow-link "${WORKSPACE_DIR}/." ${CONTAINER_NAME}:/home/user/workspace
  docker exec --user root ${CONTAINER_NAME} chown -R user:user /home/user
}

docker_exec () {
  docker exec --user user --workdir /home/user/workspace ${CONTAINER_NAME} "${@}"
}

run () {
  if [ "${LOCAL}" = true ]; then
    notif "Running local"
    # shellcheck disable=SC2086
    "${@}"
  else
    notif "Running in docker"
    docker_exec "${@}"
  fi
}
