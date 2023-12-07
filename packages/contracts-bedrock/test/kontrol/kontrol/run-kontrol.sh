#!/bin/bash
set -u

#####################
# Support Functions #
#####################
blank_line() { echo '' >&2 ; }
notif() { echo "== $0: $@" >&2 ; }

#############
# Variables #
#############
export FOUNDRY_PROFILE=kontrol
export CONTAINER_NAME=kontrol-tests
export KONTROL_RELEASE=$(cat .kontrolrc)

# Set Script Directory Variables <root>/packages/contracts-bedrock/test/kontrol/kontrol
SCRIPT_HOME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
notif "Script Home: $SCRIPT_HOME"
blank_line

# Set Run Directory <root>/packages/contracts-bedrock 
WORKSPACE_DIR=$( cd $SCRIPT_HOME/../../.. >/dev/null 2>&1 && pwd )
notif "Run Directory: $WORKSPACE_DIR"
blank_line

#############
# Functions #
#############
kontrol_build() {
    notif "Kontrol Build"
    docker_exec kontrol build                     \
                        --verbose                 \
                        --require ${lemmas}       \
                        --module-import ${module} \
                        ${rekompile}
            }

kontrol_prove() {
    notif "Kontrol Prove"
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
    docker run                        \
      --name ${CONTAINER_NAME}        \
      --rm                            \
      --interactive                   \
      --tty                           \
      --detach                        \
      --workdir /home/user/workspace  \
    runtimeverificationinc/kontrol:ubuntu-jammy-${KONTROL_RELEASE}

    # Copy test content to container
    docker cp --follow-link $WORKSPACE_DIR/. ${CONTAINER_NAME}:/home/user/workspace
    docker exec --user root ${CONTAINER_NAME} chown -R user:user /home/user
    # Install pnpm
    docker exec ${CONTAINER_NAME} curl -fsSL https://get.pnpm.io/install.sh | sh - 
}

docker_exec () {
    docker exec --workdir /home/user/workspace ${CONTAINER_NAME} $@
}

# Define the function to run on failure
on_failure() {
  notif "Something went wrong. Running cleanup..."
  notif "Creating Tar of Proof Results"
  docker_exec tar -czvf results.tar.gz kout/proofs
  
  notif "Copying Tests Results to Host"
  blank_line
  docker cp ${CONTAINER_NAME}:/home/user/workspace/results.tar.gz .
  notif "Stopping Docker Container"
  blank_line

  notif "Dump Logs"
  LOG_FILE="run-kontrol-$(date +'%Y-%m-%d-%H-%M-%S').log"
  docker logs ${CONTAINER_NAME} > $LOG_FILE

  notif "Stopping Docker Container"
  docker stop ${CONTAINER_NAME}
  blank_line

  notif "Cleanup complete."
  blank_line
  return 1
}

# Set up the trap to run the function on failure
trap on_failure ERR

#########################
# kontrol build options #
#########################
# NOTE: This script has a recurring pattern of setting and unsetting variables, such as `rekompile`. Such a pattern is intended for easy use while locally developing and executing the proofs via this script. Comment/uncomment the empty assignment to activate/deactivate the corresponding flag
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
start_docker

kontrol_build
kontrol_prove

blank_line
notif "DONE"
blank_line