#!/bin/bash
set -euo pipefail

#####################
# Support Functions #
#####################
blank_line() { echo '' >&2 ; }
notif() { echo "== $0: $*" >&2 ; }

#############
# Variables #
#############
# Set Script Directory Variables <root>/packages/contracts-bedrock/test/kontrol/kontrol
SCRIPT_HOME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
notif "Script Home: $SCRIPT_HOME"
blank_line

# Set Run Directory <root>/packages/contracts-bedrock
WORKSPACE_DIR=$( cd "${SCRIPT_HOME}/../../.." >/dev/null 2>&1 && pwd )
notif "Run Directory: ${WORKSPACE_DIR}"
blank_line

export FOUNDRY_PROFILE=kontrol
export CONTAINER_NAME=kontrol-tests
KONTROLRC=$(cat "${WORKSPACE_DIR}/../../.kontrolrc")
export KONTROL_RELEASE=${KONTROLRC}


#############
# Functions #
#############
kontrol_build() {
    notif "Kontrol Build"
    # shellcheck disable=SC2086
    docker_exec kontrol build                       \
                        --verbose                   \
                        --require ${lemmas}         \
                        --module-import ${module}   \
                        ${rekompile}
            }

kontrol_prove() {
    notif "Kontrol Prove"
    # shellcheck disable=SC2086
    docker_exec kontrol prove                              \
                        --max-depth ${max_depth}           \
                        --max-iterations ${max_iterations} \
                        --smt-timeout ${smt_timeout}       \
                        --bmc-depth ${bmc_depth}           \
                        --workers ${workers}               \
                        ${reinit}                          \
                        ${bug_report}                      \
                        ${break_on_calls}                  \
                        ${auto_abstract}                   \
                        ${tests}                           \
                        ${use_booster}
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
    docker exec --workdir /home/user/workspace ${CONTAINER_NAME} "${@}"
}

dump_log_results(){
  trap clean_docker ERR

  notif "Something went wrong. Running cleanup..."
  blank_line

  notif "Creating Tar of Proof Results"
  docker exec ${CONTAINER_NAME} tar -czvf results.tar.gz kout/proofs
  RESULTS_LOG="results-$(date +'%Y-%m-%d-%H-%M-%S').tar.gz"
  notif "Copying Tests Results to Host"
  docker cp ${CONTAINER_NAME}:/home/user/workspace/results.tar.gz "${RESULTS_LOG}"
  if [ -f "${RESULTS_LOG}" ]; then
    cp "${RESULTS_LOG}" kontrol-results_latest.tar.gz
  else
    notif "Results Log: ${RESULTS_LOG} not found, did not pull from container."
  fi
  blank_line

  notif "Dump RUN Logs"
  RUN_LOG="run-kontrol-$(date +'%Y-%m-%d-%H-%M-%S').log"
  docker logs ${CONTAINER_NAME} > "${RUN_LOG}"
}

clean_docker(){
  notif "Stopping Docker Container"
  docker stop ${CONTAINER_NAME}
  blank_line
}

# Define the function to run on failure
on_failure() {
  dump_log_results

  clean_docker

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
lemmas=test/kontrol/kontrol/pausability-lemmas.k
base_module=PAUSABILITY-LEMMAS
module=CounterTest:${base_module}
rekompile=--rekompile
rekompile=

#########################
# kontrol prove options #
#########################
max_depth=10000
max_iterations=10000
smt_timeout=100000
bmc_depth=10
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

#########################################
# List of tests to symbolically execute #
#########################################
tests=""
tests+="--match-test CounterTest.test_SetNumber "

#############
# RUN TESTS #
#############

# Is old docker container running?
if [ "$(docker ps -q -f name=${CONTAINER_NAME})" ]; then
    # Stop old docker container
    notif "Stopping old docker container"
    clean_docker
    blank_line
fi

start_docker

kontrol_build
kontrol_prove

dump_log_results

clean_docker

blank_line
notif "DONE"
blank_line
