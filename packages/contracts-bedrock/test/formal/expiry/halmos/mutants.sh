#!/usr/bin/env bash
# Mutation check for the Halmos expiry suite. For each mutant: back up the contract sources, apply one sed edit,
# run the designated checks, and REQUIRE every one of them to FAIL with a valid counterexample. Any surviving
# mutant, sed that does not apply, halmos error/timeout, or missing result makes the script exit nonzero. The
# sources are restored from the backups after each mutant and on exit (also on interrupt).
#   HALMOS  patched halmos (see run.sh); needed for the refund mutants.
#   ONLY    optional regex: run only the mutants whose name matches.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here/../../../.." # packages/contracts-bedrock
export FOUNDRY_SRC=test/formal/expiry/halmos FOUNDRY_TEST=test/formal/expiry/halmos FOUNDRY_SCRIPT=test/formal/expiry/halmos
export FOUNDRY_OUT=test/formal/expiry/halmos/out FOUNDRY_CACHE_PATH=test/formal/expiry/halmos/cache
H="${HALMOS:-halmos}"
HALMOS_WRAP="${HALMOS_WRAP:-}" # optional command prefix (e.g. a memory cap), see run.sh

L2=src/L2/L2ToL2CrossDomainMessenger.sol
L1=src/L1/L1CrossDomainMessenger.sol
CDM=src/universal/CrossDomainMessenger.sol
BR=src/L2/SuperchainETHBridge.sol
L2CDMSRC=src/L2/L2CrossDomainMessenger.sol
TC=src/libraries/TransientContext.sol
EXP=src/L2/UndeliveredMessageExporter.sol
ENC=src/libraries/Encoding.sol
FILES="$L2 $L1 $CDM $BR $L2CDMSRC $TC $EXP $ENC"
bak="$(mktemp -d)"
for f in $FILES; do cp "$f" "$bak/$(basename "$f")"; done
restore() { for f in $FILES; do cp "$bak/$(basename "$f")" "$f"; done; }
trap restore EXIT

survivors=0
# run NAME FILE SED CONTRACT "check_a check_b ..."
run() {
  local name=$1 file=$2 expr=$3 contract=$4 checks=$5
  if [ -n "${ONLY:-}" ] && ! [[ $name =~ $ONLY ]]; then return; fi
  restore
  sed -i.mut "$expr" "$file" && rm -f "$file.mut"
  if cmp -s "$file" "$bak/$(basename "$file")"; then echo "BAD $name: sed did not apply"; survivors=$((survivors + 1)); return; fi
  local regex
  regex="($(echo "$checks" | sed 's/^check_//; s/ check_/|/g'))\\("
  local json; json="$(mktemp)"
  # Force a full rebuild: test contracts embed creation code (`new X()`), and an incremental build can miss them.
  # shellcheck disable=SC2086
  if ! $HALMOS_WRAP forge build --force >/dev/null 2>&1; then
    echo "BAD $name: mutant does not compile"; survivors=$((survivors + 1)); return
  fi
  set +e
  # shellcheck disable=SC2086
  $HALMOS_WRAP "$H" --forge-build-out test/formal/expiry/halmos/out --no-status --default-bytes-lengths 0,32,100 \
    --match-contract "^${contract}\$" --match-test "$regex" --json-output "$json" >/dev/null 2>&1
  local code=$?
  set -e
  local -a check_args
  read -ra check_args <<<"$checks"
  python3 - "$name" "$contract" "$json" "$code" "${check_args[@]}" <<'EOF' || survivors=$((survivors + 1))
import json, sys
name, contract, path, code, checks = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4]), sys.argv[5:]
if code != 1:
    print(f"BAD {name}: halmos process exited {code} (expected 1: every designated check fails)"); sys.exit(1)
try:
    res = json.load(open(path))["test_results"]
except Exception as e:
    print(f"BAD {name}: no halmos JSON ({e})"); sys.exit(1)
rows = {r["name"].split("(")[0]: r for k, v in res.items() if k.split(":")[-1] == contract for r in (v or [])}
bad = []
for c in checks:
    r = rows.get(c)
    if r is None:
        bad.append(f"{c}: no result")
    # A kill needs one valid counterexample. Stuck paths are tolerated HERE (not in run.sh): a mutant can open code
    # paths halmos cannot finish (e.g. M20 lets the messenger call itself with symbolic calldata).
    elif not (r["exitcode"] == 1 and (r["num_models"] or 0) > 0 and any(m.get("is_valid") for m in r["models"] or [])):
        bad.append(f"{c}: exitcode={r['exitcode']} (survived or errored)")
if bad:
    print(f"BAD {name}: " + "; ".join(bad)); sys.exit(1)
print(f"killed {name}: {' '.join(checks)}")
EOF
  rm -f "$json"
}

# run_forge NAME FILE SED TEST_PATH TEST_NAME: a mutant this suite cannot see (e.g. emitted events: halmos 0.3.3 has
# no recordLogs), required to be killed by an existing forge test instead (default foundry profile).
run_forge() {
  local name=$1 file=$2 expr=$3 path=$4 test=$5
  if [ -n "${ONLY:-}" ] && ! [[ $name =~ $ONLY ]]; then return; fi
  local log="$bak/$name.log"
  # Baseline: the designated test must PASS on the unmutated code (so setUp works).
  restore
  set +e
  ( unset FOUNDRY_SRC FOUNDRY_TEST FOUNDRY_SCRIPT FOUNDRY_OUT FOUNDRY_CACHE_PATH
    # shellcheck disable=SC2086
    $HALMOS_WRAP forge test --match-path "$path" --match-test "$test" >"$log.base" 2>&1 )
  local base=$?
  set -e
  if [ "$base" -ne 0 ] || ! grep -q "\[PASS\] $test(" "$log.base"; then
    echo "BAD $name: baseline forge test $test does not pass (exit $base)"; survivors=$((survivors + 1)); return
  fi
  sed -i.mut "$expr" "$file" && rm -f "$file.mut"
  if cmp -s "$file" "$bak/$(basename "$file")"; then echo "BAD $name: sed did not apply"; survivors=$((survivors + 1)); return; fi
  set +e
  ( unset FOUNDRY_SRC FOUNDRY_TEST FOUNDRY_SCRIPT FOUNDRY_OUT FOUNDRY_CACHE_PATH
    # shellcheck disable=SC2086
    $HALMOS_WRAP forge test --match-path "$path" --match-test "$test" >"$log" 2>&1 )
  local code=$?
  set -e
  # Killed only if the designated test itself failed after a successful setUp (not a setUp/compile failure).
  if [ "$code" -ne 0 ] && grep -qE "^\[FAIL.*\] $test\(" "$log" && ! grep -q "setUp()" "$log" \
      && ! grep -q "Compiler run failed" "$log"; then
    echo "killed $name (forge): $path::$test"
  else
    echo "BAD $name: forge test $test did not fail as required (exit $code)"; survivors=$((survivors + 1))
  fi
}

# --- killed elsewhere (forge)
run_forge X1_event_nonce_plus1 $L2 's/emit SentMessage(_destination, _target, nonce, msg.sender, _message);/emit SentMessage(_destination, _target, nonce + 1, msg.sender, _message);/' \
  test/L2/L2ToL2CrossDomainMessenger.t.sol testFuzz_sendMessage_succeeds
run_forge X1b_event_sender_origin $L2 's/emit SentMessage(_destination, _target, nonce, msg.sender, _message);/emit SentMessage(_destination, _target, nonce, tx.origin, _message);/' \
  test/L2/L2ToL2CrossDomainMessenger.t.sol testFuzz_sendMessage_succeeds

# --- L2ToL2CrossDomainMessenger
run M1_window_lt $L2 's/if (_undeliveredAt <= sentAt + EXPIRY_PERIOD)/if (_undeliveredAt < sentAt + EXPIRY_PERIOD)/' \
  L2ToL2ExpiryHalmos "check_expire_iff check_expire_iff_unbounded check_expire_boundary"
run M2_relay_skips_unsafe_check $L2 's/^        if (_isUnsafeTarget(target)) revert L2ToL2CrossDomainMessenger_MessageTargetUnsafe();$//' \
  L2ToL2ExpiryHalmos "check_UnsafeTargetRule_relay check_UnsafeTargetRule_relay_l2cdm check_UnsafeTargetRule_relay_passer check_OnlyExportReachesL1_relay_l2cdm check_OnlyExportReachesL1_relay_passer"
run M3_send_skips_unsafe_check $L2 's/^        if (_isUnsafeTarget(_target)) revert L2ToL2CrossDomainMessenger_MessageTargetUnsafe();$//' \
  L2ToL2ExpiryHalmos "check_UnsafeTargetRule_send check_UnsafeTargetRule_send_passer"
run M3b_send_no23 $L2 's/if (_target == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert MessageTargetL2ToL2CrossDomainMessenger();//' \
  L2ToL2ExpiryHalmos "check_UnsafeTargetRule_send"
run M3c_unsafe_drops_passer $L2 's/ || _target == Predeploys.L2_TO_L1_MESSAGE_PASSER;/;/' \
  L2ToL2ExpiryHalmos "check_UnsafeTargetRule_send_passer check_UnsafeTargetRule_relay_passer check_OnlyExportReachesL1_relay_passer"
run M7_expire_no_xdomain_check $L2 's/|| ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).xDomainMessageSender()/|| address(0)/' \
  L2ToL2ExpiryHalmos "check_expire_iff check_expire_iff_unbounded"
# --- UndeliveredMessageExporter
run M4_export_dest_is_source $EXP 's/_destination: block.chainid,/_destination: _source,/' \
  ExporterExpiryHalmos "check_export_binding check_exporter_anyCalldata_onlyExportPayload"
run M5_export_reads_wrong_map $EXP 's/.successfulMessages(messageHash_)) {/.expiredMessages(messageHash_)) {/' \
  ExporterExpiryHalmos "check_export_binding"
run M6_export_ts0 $EXP 's/(messageHash_, block.timestamp)/(messageHash_, 0)/' \
  ExporterExpiryHalmos "check_export_binding check_exporter_anyCalldata_onlyExportPayload"
run M15_export_wrong_target $EXP 's/_target: _sourceMessenger,/_target: address(uint160(_sourceMessenger) + 1),/' \
  ExporterExpiryHalmos "check_export_binding check_exporter_anyCalldata_onlyExportPayload"
run M13_send_timestamp_1 $L2 's/sentMessageTimestamps\[messageHash_\] = block.timestamp;/sentMessageTimestamps[messageHash_] = 1;/' \
  L2ToL2ExpiryHalmos "check_send_effects_and_frame"
run M14_relay_sets_expired $L2 's/        successfulMessages\[messageHash\] = true;/        successfulMessages[messageHash] = true; expiredMessages[messageHash] = true;/' \
  L2ToL2ExpiryHalmos "check_relay_effects_and_frame"
run M16_send_no_sentMessages $L2 's/sentMessages\[nonce\] = messageHash_;//' \
  L2ToL2ExpiryHalmos "check_send_effects_and_frame"
run M17_expire_sets_successful $L2 's/expiredMessages\[_messageHash\] = true;/expiredMessages[_messageHash] = true; successfulMessages[_messageHash] = true;/' \
  L2ToL2ExpiryHalmos "check_expire_iff"
run M24_relay_no_mark_before_call $L2 's/        successfulMessages\[messageHash\] = true;//' \
  L2ToL2ExpiryHalmos "check_relay_effects_and_frame check_OnlyExportReachesL1_relay_reentrant"
run M25_relay_swallows_target_revert $L2 's/                revert(add(32, returnData_), mload(returnData_))//' \
  L2ToL2ExpiryHalmos "check_relay_delivery_value_context_failure"
run M26_relay_not_nonReentrant $L2 '/^        nonReentrant$/d' \
  L2ToL2ExpiryHalmos "check_relay_delivery_value_context_failure"
run M27_entered_not_cleared $TC 's/tstore(ENTERED_SLOT, 0)/tstore(ENTERED_SLOT, 1)/' \
  L2ToL2ExpiryHalmos "check_relay_delivery_value_context_failure"
run M36_no_nested_relay_guard $TC 's/        if (_entered()) revert ReentrantCall();//' \
  L2ToL2ExpiryHalmos "check_OnlyExportReachesL1_relay_reentrant"
run M28_context_sender_is_target $L2 's/_storeMessageMetadata(source, sender);/_storeMessageMetadata(source, target);/' \
  L2ToL2ExpiryHalmos "check_relay_delivery_value_context_failure"
run M29_relay_drops_value $L2 's/target.call{ value: msg.value }(message)/target.call(message)/' \
  L2ToL2ExpiryHalmos "check_relay_delivery_value_context_failure"
# --- L1CrossDomainMessenger / CrossDomainMessenger / L2CrossDomainMessenger
run M35_cdm_unauthorized_marks_failed $CDM 's/            require(failedMessages\[versionedHash\], "CrossDomainMessenger: message cannot be replayed");/            if (!failedMessages[versionedHash]) { failedMessages[versionedHash] = true; return; }/' \
  L1CDMExpiryHalmos "check_L1_relayGate_and_delivery"
run M35b_cdm_unauthorized_marks_failed_L2 $CDM 's/            require(failedMessages\[versionedHash\], "CrossDomainMessenger: message cannot be replayed");/            if (!failedMessages[versionedHash]) { failedMessages[versionedHash] = true; return; }/' \
  L2CDMGateHalmos "check_L2_relayGate_and_delivery"
run M37_encoding_drops_sender $ENC '/function encodeCrossDomainMessageV1/,/^    }/ s/^            _sender,$/            address(uint160(_sender) \& 0),/' \
  L1CDMExpiryHalmos "check_L1_failedEntryNotReplayableWithAlteredField"
run M33_l1_no_interop_gate $L1 's/        if (!systemConfig.isFeatureEnabled(Features.INTEROP)) revert L1CrossDomainMessenger_NotInteropMessenger();//' \
  L1CDMExpiryHalmos "check_relayUndelivered_iff_and_deposit"
run M34_l1_trusts_l2tol2 $L1 's/!= Predeploys.UNDELIVERED_MESSAGE_EXPORTER/!= Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER/' \
  L1CDMExpiryHalmos "check_relayUndelivered_iff_and_deposit check_relayUndelivered_rejectsL2ToL2AsSender"
run M8_l1_no_lockbox $L1 's/|| !portal.ethLockbox().authorizedPortals(callerPortal)//' \
  L1CDMExpiryHalmos "check_relayUndelivered_iff_and_deposit"
run M9_l1_gas $L1 's/EXPIRE_MESSAGE_GAS_LIMIT = 100_000/EXPIRE_MESSAGE_GAS_LIMIT = 100_001/' \
  L1CDMExpiryHalmos "check_relayUndelivered_iff_and_deposit"
run M20_l1_self_not_unsafe $L1 's/return _target == address(this) || _target == address(portal);/return _target == address(portal);/' \
  L1CDMExpiryHalmos "check_L1_relayMessage_rejectsSelfAndPortalTargets"
run M21_cdm_sender_not_reset $CDM 's/^        xDomainMsgSender = Constants.DEFAULT_L2_SENDER;$//' \
  L1CDMExpiryHalmos "check_L1_relayGate_and_delivery"
run M23_cdm_sender_is_origin $CDM 's/messageNonce(), msg.sender, _target, msg.value/messageNonce(), tx.origin, _target, msg.value/' \
  L1CDMExpiryHalmos "check_L1_sendMessage_senderFieldIsCaller check_relayUndelivered_iff_and_deposit"
run X2_l1_gate_ignores_l2Sender $L1 's/return msg.sender == address(portal) \&\& portal.l2Sender() == address(otherMessenger);/return msg.sender == address(portal);/' \
  L1CDMExpiryHalmos "check_L1_relayGate_and_delivery"
run X3a_cdm_no_replay_gate_L1 $CDM 's/            require(failedMessages\[versionedHash\], "CrossDomainMessenger: message cannot be replayed");//' \
  L1CDMExpiryHalmos "check_L1_relayGate_and_delivery"
run X3b_cdm_no_replay_gate_L2 $CDM 's/            require(failedMessages\[versionedHash\], "CrossDomainMessenger: message cannot be replayed");//' \
  L2CDMGateHalmos "check_L2_relayGate_and_delivery"
run X2b_l2_gate_any_caller $L2CDMSRC 's/return AddressAliasHelper.undoL1ToL2Alias(msg.sender) == address(otherMessenger);/return msg.sender != address(0);/' \
  L2CDMGateHalmos "check_L2_relayGate_and_delivery"
run M30_cdm_relay_drops_value $CDM 's/SafeCall.call(_target, gasleft() - RELAY_RESERVED_GAS, _value, _message)/SafeCall.call(_target, gasleft() - RELAY_RESERVED_GAS, 0, _message)/' \
  L1CDMExpiryHalmos "check_L1_relayGate_and_delivery"
run M31_cdm_send_drops_value $CDM 's/            _value: msg.value,/            _value: 0,/' \
  L1CDMExpiryHalmos "check_L1_sendMessage_senderFieldIsCaller"
run M32_l2cdm_self_not_unsafe $L2CDMSRC 's/return _target == address(this) || _target == address(Predeploys.L2_TO_L1_MESSAGE_PASSER);/return _target == address(otherMessenger) || _target == address(Predeploys.L2_TO_L1_MESSAGE_PASSER);/' \
  L2CDMGateHalmos "check_L2_relayMessage_rejectsSelfAndPasser"
# --- SuperchainETHBridge
run M10_refund_no_refunded_check $BR 's/if (refunded\[messageHash\]) revert SuperchainETHBridge_AlreadyRefunded();//' \
  RefundExpiryHalmos "check_refund_iff_effects_singleUse"
run M11_refund_pays_to $BR 's/new SafeSend{ value: _amount }(payable(_from));/new SafeSend{ value: _amount }(payable(_to));/' \
  RefundExpiryHalmos "check_refund_iff_effects_singleUse check_sendETH_then_refund"
run M12_refund_no_mark $BR 's/        refunded\[messageHash\] = true;//' \
  RefundExpiryHalmos "check_refund_iff_effects_singleUse"
run M18_refund_hash_nonce_plus1 $BR 's/            _nonce: _nonce,/            _nonce: _nonce + 1,/' \
  RefundExpiryHalmos "check_refund_iff_effects_singleUse check_sendETH_then_refund"
run M19_refund_hash_sender_from $BR 's/            _sender: address(this),/            _sender: _from,/' \
  RefundExpiryHalmos "check_refund_iff_effects_singleUse check_sendETH_then_refund"

restore
forge build >/dev/null 2>&1 || true
if [ "$survivors" -ne 0 ]; then echo "$survivors mutant(s) not killed as required"; exit 1; fi
echo "all mutants killed by their designated checks"
