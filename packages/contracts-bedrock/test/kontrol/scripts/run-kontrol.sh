#!/bin/bash
set -euo pipefail

#####################
# Support Functions #
#####################
blank_line() { echo '' >&2 ; }
notif() { echo "== $0: $*" >&2 ; }
usage() {
  echo "Usage: $0 [-h|--help] [container|local|dev]" 1>&2
  echo "Options:" 1>&2
  echo "  -h, --help         Display this help message." 1>&2
  echo "  container          Run tests in docker container. Reproduce CI execution. (Default)" 1>&2
  echo "  local              Run locally, enforces CI Registered .kontrolrc Kontrol version for better reproducibility. (Recommended)" 1>&2
  echo "  dev                Run locally, do NOT enforce CI registered Kontrol version (Recomended w/ greater kup & kontrol experience)" 1>&2
  exit 0
}

#############
# Variables #
#############
# Set Script Directory Variables <root>/packages/contracts-bedrock/test/kontrol
SCRIPT_HOME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
notif "Script Home: $SCRIPT_HOME"
blank_line

# Set Run Directory <root>/packages/contracts-bedrock
WORKSPACE_DIR=$( cd "${SCRIPT_HOME}/../../.." >/dev/null 2>&1 && pwd )
notif "Run Directory: ${WORKSPACE_DIR}"
blank_line

export FOUNDRY_PROFILE=kprove
export CONTAINER_NAME=kontrol-tests
KONTROLRC=$(jq -r .kontrol < "${WORKSPACE_DIR}/../../versions.json")
export KONTROL_RELEASE=${KONTROLRC}
export LOCAL=false

#######################
# Check for arguments #
#######################
if [ $# -gt 1 ]; then
  usage
else
  if [ $# -eq 0 ] || [ "$1" == "container" ]; then
    notif "Running in docker container (DEFAULT)"
    blank_line
    export LOCAL=false
    [ $# -gt 1 ] && shift
  elif [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
    usage
  elif [ "$1" == "local" ]; then
    notif "Running with LOCAL install, .kontrolrc CI version ENFORCED"
    if [ "$(kontrol version | awk -F': ' '{print$2}')" == "${KONTROLRC}" ]; then
      notif "Kontrol version matches ${KONTROLRC}"
      blank_line
      export LOCAL=true
      shift
      pushd "${WORKSPACE_DIR}" > /dev/null
    else
      notif "Kontrol version does NOT match ${KONTROLRC}"
      notif "Please run 'kup install kontrol --version v${KONTROLRC}'"
      blank_line
      exit 1
    fi
  elif [ "$1" == "dev" ]; then
    notif "Running with LOCAL install, IGNORING .kontrolrc version"
    blank_line
    export LOCAL=true
    shift
    pushd "${WORKSPACE_DIR}" > /dev/null
  else
    # Unexpected argument passed
    usage
  fi
fi

#############
# Functions #
#############
kontrol_build() {
    notif "Kontrol Build"
    # shellcheck disable=SC2086
    run kontrol build                       \
                        --verbose                   \
                        --require ${lemmas}         \
                        --module-import ${module}   \
                        ${rekompile}
            }

kontrol_prove() {
    notif "Kontrol Prove"
    # shellcheck disable=SC2086
    run kontrol prove                              \
                        --max-depth ${max_depth}           \
                        --max-iterations ${max_iterations} \
                        --smt-timeout ${smt_timeout}       \
                        --workers ${workers}               \
                        ${reinit}                          \
                        ${bug_report}                      \
                        ${break_on_calls}                  \
                        ${auto_abstract}                   \
                        ${tests}                           \
                        ${use_booster}                     \
                        --init-node-from ${state_diff}
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

dump_log_results(){
  trap clean_docker ERR
    RESULTS_FILE="results-$(date +'%Y-%m-%d-%H-%M-%S').tar.gz"
    LOG_PATH="test/kontrol/logs"
    RESULTS_LOG="${LOG_PATH}/${RESULTS_FILE}"

    if [ ! -d ${LOG_PATH} ]; then
      mkdir ${LOG_PATH}
    fi

    notif "Generating Results Log: ${LOG_PATH}"
    blank_line

    run tar -czvf results.tar.gz kout-proofs/ > /dev/null 2>&1
    if [ "${LOCAL}" = true ]; then
      mv results.tar.gz "${RESULTS_LOG}"
    else
      docker cp ${CONTAINER_NAME}:/home/user/workspace/results.tar.gz "${RESULTS_LOG}"
    fi
    if [ -f "${RESULTS_LOG}" ]; then
      cp "${RESULTS_LOG}" "${LOG_PATH}/kontrol-results_latest.tar.gz"
    else
      notif "Results Log: ${RESULTS_LOG} not found, skipping.."
      blank_line
    fi
    # Report where the file was generated and placed
    notif "Results Log: $(dirname "${RESULTS_LOG}") generated"

    if [ "${LOCAL}" = false ]; then
      notif "Results Log: ${RESULTS_LOG} generated"
      blank_line
      RUN_LOG="run-kontrol-$(date +'%Y-%m-%d-%H-%M-%S').log"
      docker logs ${CONTAINER_NAME} > "${LOG_PATH}/${RUN_LOG}"
    fi
}

clean_docker(){
    notif "Stopping Docker Container"
    docker stop ${CONTAINER_NAME}
    blank_line
}

# Define the function to run on failure
on_failure() {
    dump_log_results

    if [ "${LOCAL}" = false ]; then
      clean_docker
    fi

    notif "Cleanup complete."
    blank_line
    exit 1
}

# Set up the trap to run the function on failure
trap on_failure ERR INT

#########################
# kontrol build options #
#########################
# NOTE: This script has a recurring pattern of setting and unsetting variables,
# such as `rekompile`. Such a pattern is intended for easy use while locally
# developing and executing the proofs via this script. Comment/uncomment the
# empty assignment to activate/deactivate the corresponding flag
lemmas=test/kontrol/pausability-lemmas.k
base_module=PAUSABILITY-LEMMAS
module=OptimismPortalKontrol:${base_module}
rekompile=--rekompile
rekompile=
regen=--regen
# shellcheck disable=SC2034
regen=

#########################
# kontrol prove options #
#########################
max_depth=1000000
max_iterations=1000000
smt_timeout=100000
workers=2
reinit=--reinit
reinit=
break_on_calls=--no-break-on-calls
# break_on_calls=
auto_abstract=--auto-abstract-gas
# auto_abstract=
bug_report=--bug-report
bug_report=
use_booster=--use-booster
# use_booster=
state_diff="./snapshots/state-diff/Kontrol-Deploy.json"

#########################################
# List of tests to symbolically execute #
#########################################
tests=""
#tests+="--match-test OptimismPortalKontrol.prove_proveWithdrawalTransaction_paused "
tests+="--match-test OptimismPortalKontrol.prove_finalizeWithdrawalTransaction_paused "
tests+="--match-test L1CrossDomainMessengerKontrol.prove_relayMessage_paused "

#############
# RUN TESTS #
#############
if [ "${LOCAL}" == false ]; then
  # Is old docker container running?
  if [ "$(docker ps -q -f name=${CONTAINER_NAME})" ]; then
      # Stop old docker container
      notif "Stopping old docker container"
      clean_docker
      blank_line
  fi
  start_docker
fi

kontrol_build
kontrol_prove

dump_log_results

if [ "${LOCAL}" == false ]; then
    notif "Stopping docker container"
    clean_docker
fi

blank_line
notif "DONE"
blank_line
