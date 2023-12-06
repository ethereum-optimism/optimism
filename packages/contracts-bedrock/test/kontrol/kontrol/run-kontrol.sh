#!/bin/bash

set -euxo pipefail

#####################
# Support Functions #
#####################
blank_line() { echo '' >&2 ; }
notif() { echo "== $0: $@" >&2 ; }

###########
# Globals #
###########
export FOUNDRY_PROFILE=kontrol

# Set Script Directory Variables <root>/packages/contracts-bedrock/test/kontrol/kontrol
SCRIPT_HOME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
notif "Script Home: $SCRIPT_HOME"
blank_line

# Set Run Directory <root>/packages/contracts-bedrock 
RUN_DIR="$( cd "$( dirname "$SCRIPT_HOME/../../.." )" >/dev/null 2>&1 && pwd )"
notif "Run Directory: $RUN_DIR"
blank_line

# Set Log Directory <root>/packages/contracts-bedrock/test/kontrol/kontrol/logs
LOG_DIRECTORY=$SCRIPT_HOME/logs
notif "Log Directory: $LOG_DIRECTORY"
blank_line

if [ ! -d $LOG_DIRECTORY ] ; then
  mkdir $LOG_DIRECTORY
fi

LOG_FILE="run-kontrol-$(date +'%Y-%m-%d-%H-%M-%S').log"
notif "Logging to $LOG_DIRECTORY/$LOG_FILE"
blank_line
exec > >(tee -i $LOG_DIRECTORY/$LOG_FILE)
exec 2>&1

#############
# Functions #
#############
kontrol_build() {
    notif "Kontrol Build"
    kontrol build                     \
            --verbose                 \
            --require ${lemmas}       \
            --module-import ${module} \
            ${rekompile}
}

kontrol_prove() {
    notif "Kontrol Prove"
    kontrol prove                              \
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
pushd $RUN_DIR
notif "Running Kontrol Build"
blank_line
kontrol_build

notif "Running Kontrol Prove"
blank_line
kontrol_prove

blank_line
notif "DONE"
blank_line
popd 2>&1 > /dev/null