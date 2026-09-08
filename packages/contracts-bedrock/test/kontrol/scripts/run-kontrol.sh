#!/bin/bash
set -Eeuo pipefail

export FOUNDRY_PROFILE=kprove

SCRIPT_HOME="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_HOME/common.sh"
case "${KONTROL_CALLER_ONLY:-false}" in
  false) ;;
  true)
    [ "${KONTROL_STRICT:-false}" = true ] || {
      echo "KONTROL_CALLER_ONLY requires strict mode" >&2; exit 1;
    } ;;
  *) echo "KONTROL_CALLER_ONLY must be true or false" >&2; exit 1 ;;
esac
if [ "${KONTROL_STRICT:-false}" = true ]; then
  # The previous suite's auto-removal can still hold its container name.
  export CONTAINER_NAME="${CONTAINER_NAME}-withdrawal-$$"
fi
export RUN_KONTROL=true
parse_args "$@"

#############
# Functions #
#############
kontrol_build() {
  notif "Kontrol Build"
  local build_command=(kontrol build)
  # Kontrol flattens contract-qualified imports into its shared main module.
  if [ "${KONTROL_STRICT:-false}" != true ]; then
    build_command+=(--require "$lemmas" --module-import "$module")
  fi
  # shellcheck disable=SC2086
  run "${build_command[@]}" \
    --no-metadata \
    ${rekompile} \
    ${regen}
  return $?
}

kontrol_prove() {
  notif "Kontrol Prove: workers=$workers selectors=${test_list[*]}"
  local model_args=(--init-node-from-diff "$state_diff" --assume-defined --no-stack-checks)
  local prove_command=(kontrol prove)
  local methods_status=0
  local maintenance_rate=16
  local rpc_command='kore-rpc-booster --equation-max-recursion 100 --equation-max-iterations 1000'
  # Withdrawal fixtures retain production deployment bytecode and stack checks.
  if [ "${KONTROL_STRICT:-false}" = true ]; then
    # An interrupt skips Kontrol's final save; retain every completed proof step.
    maintenance_rate=1
    model_args=(--init-node-from-diff "$state_diff" --reinit --schedule CANCUN --use-gas)
    # Keep post-execution simplification to eliminate infeasible symbolic dispatch branches.
    # Booster checks branch coverage; retain legacy fallback for stuck or aborted execution.
    rpc_command+=' --fallback-on Stuck,Aborted'
    # Bound proving inside the container so the host can still collect its saved graphs.
    local proof_timeout=60m
    if [ "${KONTROL_CALLER_ONLY:-false}" = true ]; then
      proof_timeout=15m
      max_depth=1000
    fi
    prove_command=(timeout --signal=INT --kill-after=30s "$proof_timeout" kontrol prove --verbose)
  else
    rpc_command+=' --no-post-exec-simplify'
  fi
  # shellcheck disable=SC2086
  run "${prove_command[@]}" \
    --max-depth $max_depth \
    --max-iterations $max_iterations \
    --smt-timeout $smt_timeout \
    --workers $workers \
    $reinit \
    $bug_report \
    $break_on_calls \
    $break_every_step \
    $tests \
    "${model_args[@]}" \
    --kore-rpc-command "$rpc_command" \
    --xml-test-report \
    --maintenance-rate "$maintenance_rate" \
    --symbolic-caller \
    --no-log-rewrites \
    --smt-timeout 16000 \
    --smt-retry-limit 0 \
    --remove-old-proofs || methods_status=$?
  if [ "${KONTROL_CALLER_ONLY:-false}" = true ]; then
    notif "CALLER DIAGNOSTIC: exit=$methods_status; the full suite was not run"
    return "$methods_status"
  fi
  return "$methods_status"
}

get_log_results() {
  RESULTS_FILE="results-$(date +'%Y-%m-%d-%H-%M-%S').tar.gz"
  LOG_PATH="${KONTROL_LOG_DIR:-test/kontrol/logs}"
  RESULTS_LOG="$LOG_PATH/$RESULTS_FILE"

  if [ ! -d $LOG_PATH ]; then
    mkdir -p "$LOG_PATH"
  fi

  notif "Generating Results Log: $RESULTS_LOG"

  run tar -czf results.tar.gz kout-proofs/ > /dev/null 2>&1
  if [ "$LOCAL" = true ]; then
    mv results.tar.gz "$RESULTS_LOG"
  else
    docker cp "$CONTAINER_NAME:/home/user/workspace/results.tar.gz" "$RESULTS_LOG"
    # Check if kontrol_prove_report.xml exists in the container and copy it out if it does
    if docker exec "$CONTAINER_NAME" test -f /home/user/workspace/kontrol_prove_report.xml; then
      docker cp "$CONTAINER_NAME:/home/user/workspace/kontrol_prove_report.xml" "$LOG_PATH/kontrol_prove_report.xml"
      notif "Copied kontrol_prove_report.xml to $LOG_PATH"
    else
      notif "kontrol_prove_report.xml not found in container"
    fi
    tar -xzf "$RESULTS_LOG" > /dev/null 2>&1
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
    # Expand the tar folder to kout-proofs for Summary Results and caching
    tar -xzf "$RESULTS_LOG" -C "$WORKSPACE_DIR" > /dev/null 2>&1
  fi
}

# Define the function to run on failure
on_failure() {
  trap - ERR INT TERM
  get_log_results || true

  if [ "$LOCAL" = false ]; then
    clean_docker
  fi

  notif "Failure Cleanup Complete."
  exit 1
}

#########################
# kontrol build options #
#########################
# NOTE: This script has a recurring pattern of setting and unsetting variables,
# such as `rekompile`. Such a pattern is intended for easy use while locally
# developing and executing the proofs via this script. Comment/uncomment the
# empty assignment to activate/deactivate the corresponding flag
lemmas=test/kontrol/pausability-lemmas.md
base_module=PAUSABILITY-LEMMAS
module=OptimismPortal2Kontrol:$base_module
rekompile=--rekompile
# rekompile=
regen=--regen
# regen=

#################################
# Tests to symbolically execute #
#################################
test_list=()
if [ "$SCRIPT_TESTS" == true ]; then
  test_list=(
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused0"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused1("
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused2"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused3"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused4"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused5"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused6"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused7"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused8"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused9"
    "OptimismPortal2Kontrol.prove_proveWithdrawalTransaction_paused10"
    "OptimismPortal2Kontrol.prove_finalizeWithdrawalTransaction_paused"
    "L1StandardBridgeKontrol.prove_finalizeBridgeERC20_paused"
    "L1StandardBridgeKontrol.prove_finalizeBridgeETH_paused"
    "L1ERC721BridgeKontrol.prove_finalizeBridgeERC721_paused"
    "L1CrossDomainMessengerKontrol.prove_relayMessage_paused"
  )
elif [ "$CUSTOM_TESTS" != 0 ]; then
  test_list=("${@:${CUSTOM_TESTS}}")
fi
tests=""
for test_name in "${test_list[@]}"; do
  tests+="--match-test $test_name "
done

#########################
# kontrol prove options #
#########################
max_depth=10000
max_iterations=10000
smt_timeout=100000
max_workers=${KONTROL_WORKERS:-16}
if ! [[ "$max_workers" =~ ^[1-9][0-9]*$ ]]; then
  echo "KONTROL_WORKERS must be a positive integer" >&2
  exit 1
fi
# workers is the minimum between max_workers and the length of test_list unless
# no test arguments are provided, in which case we default to max_workers
if [ "$CUSTOM_TESTS" == 0 ] && [ "$SCRIPT_TESTS" == false ]; then
  workers=${max_workers}
else
  workers=$((${#test_list[@]} > max_workers ? max_workers : ${#test_list[@]}))
fi
reinit=--reinit
reinit=
break_on_calls=--break-on-calls
break_on_calls=
break_every_step=--break-every-step
break_every_step=
bug_report=--bug-report
bug_report=
state_diff="./snapshots/state-diff/Kontrol-31337.json"

#############
# RUN TESTS #
#############
# Set up the trap to run the function on failure
trap on_failure ERR INT TERM
trap clean_docker EXIT
conditionally_start_docker

results=()

# Run kontrol_build and store the result
kontrol_build
results[0]=$?
if [ "${results[0]}" -ne 0 ]; then
  echo "Kontrol Build Failed"
  exit 1
fi

# Run kontrol_prove and store the result
LOG_PATH="${KONTROL_LOG_DIR:-test/kontrol/logs}"
mkdir -p "$LOG_PATH"
# Only the parent collects artifacts and cleans up when the logging pipeline fails.
(
  trap - ERR INT TERM
  kontrol_prove
) 2>&1 | tee "$LOG_PATH/kontrol-prove.log"
results[1]=$?
if [ "${results[1]}" -ne 0 ]; then
  echo "Kontrol Prove Failed"
  exit 2
fi

get_log_results
echo "Kontrol Passed"
notif "DONE"
