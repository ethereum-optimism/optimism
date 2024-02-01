#!/bin/bash
set -euo pipefail

export FOUNDRY_PROFILE=kprove

SCRIPT_HOME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# shellcheck source=/dev/null
source "$SCRIPT_HOME/common.sh"
parse_args "$@"

#############
# Functions #
#############
kontrol_build() {
  notif "Kontrol Build"
  # shellcheck disable=SC2086
  run kontrol build \
    --verbose \
    --require $lemmas \
    --module-import $module \
    $rekompile
}

kontrol_prove() {
  notif "Kontrol Prove"
  # shellcheck disable=SC2086
  run kontrol prove \
    --max-depth $max_depth \
    --max-iterations $max_iterations \
    --smt-timeout $smt_timeout \
    --workers $workers \
    $reinit \
    $bug_report \
    $break_on_calls \
    $auto_abstract \
    $tests \
    $use_booster \
    --init-node-from $state_diff
}

dump_log_results(){
  trap clean_docker ERR
    RESULTS_FILE="results-$(date +'%Y-%m-%d-%H-%M-%S').tar.gz"
    LOG_PATH="test/kontrol/logs"
    RESULTS_LOG="$LOG_PATH/$RESULTS_FILE"

    if [ ! -d $LOG_PATH ]; then
      mkdir $LOG_PATH
    fi

    notif "Generating Results Log: $LOG_PATH"

    run tar -czvf results.tar.gz kout-proofs/ > /dev/null 2>&1
    if [ "$LOCAL" = true ]; then
      mv results.tar.gz "$RESULTS_LOG"
    else
      docker cp "$CONTAINER_NAME:/home/user/workspace/results.tar.gz" "$RESULTS_LOG"
    fi
    if [ -f "$RESULTS_LOG" ]; then
      cp "$RESULTS_LOG" "$LOG_PATH/kontrol-results_latest.tar.gz"
    else
      notif "Results Log: $RESULTS_LOG not found, skipping.."
    fi
    # Report where the file was generated and placed
    notif "Results Log: $(dirname "$RESULTS_LOG") generated"

    if [ "$LOCAL" = false ]; then
      notif "Results Log: $RESULTS_LOG generated"
      RUN_LOG="run-kontrol-$(date +'%Y-%m-%d-%H-%M-%S').log"
      docker logs "$CONTAINER_NAME" > "$LOG_PATH/$RUN_LOG"
    fi
}

# Define the function to run on failure
on_failure() {
  dump_log_results

  if [ "$LOCAL" = false ]; then
    clean_docker
  fi

  notif "Cleanup complete."
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
module=OptimismPortalKontrol:$base_module
rekompile=--rekompile
rekompile=
regen=--regen
# shellcheck disable=SC2034
regen=

#################################
# Tests to symbolically execute #
#################################
# Missing: OptimismPortalKontrol.prove_proveWithdrawalTransaction_paused
test_list=( "OptimismPortalKontrol.prove_finalizeWithdrawalTransaction_paused" \
            "L1StandardBridgeKontrol.prove_finalizeBridgeERC20_paused" \
            "L1StandardBridgeKontrol.prove_finalizeBridgeETH_paused" \
            "L1ERC721BridgeKontrol.prove_finalizeBridgeERC721_paused" \
            "L1CrossDomainMessengerKontrol.prove_relayMessage_paused"
          )
tests=""
for test_name in "${test_list[@]}"; do
  tests+="--match-test $test_name "
done

#########################
# kontrol prove options #
#########################
max_depth=1000000
max_iterations=1000000
smt_timeout=100000
max_workers=7 # Set to 7 since the CI machine has 8 CPUs
# workers is the minimum between max_workers and the length of test_list
workers=$((${#test_list[@]}>max_workers ? max_workers : ${#test_list[@]}))
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

#############
# RUN TESTS #
#############
conditionally_start_docker

kontrol_build
kontrol_prove

dump_log_results

if [ "$LOCAL" == false ]; then
    notif "Stopping docker container"
    clean_docker
fi

notif "DONE"
