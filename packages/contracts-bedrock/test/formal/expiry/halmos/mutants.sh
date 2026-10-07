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
export FOUNDRY_OUT=halmos-out FOUNDRY_CACHE_PATH=halmos-cache
H="${HALMOS:-halmos}"

L2=src/L2/L2ToL2CrossDomainMessenger.sol
L1=src/L1/L1CrossDomainMessenger.sol
CDM=src/universal/CrossDomainMessenger.sol
BR=src/L2/SuperchainETHBridge.sol
FILES="$L2 $L1 $CDM $BR"
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
  set +e
  "$H" --forge-build-out halmos-out --no-status --default-bytes-lengths 0,32,100 \
    --match-contract "^${contract}\$" --match-test "$regex" --json-output "$json" >/dev/null 2>&1
  set -e
  python3 - "$name" "$contract" "$json" $checks <<'EOF' || survivors=$((survivors + 1))
import json, sys
name, contract, path, checks = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4:]
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
    elif not (r["exitcode"] == 1 and (r["num_models"] or 0) > 0 and any(m.get("is_valid") for m in r["models"] or [])):
        bad.append(f"{c}: exitcode={r['exitcode']} (survived or errored)")
if bad:
    print(f"BAD {name}: " + "; ".join(bad)); sys.exit(1)
print(f"killed {name}: {' '.join(checks)}")
EOF
  rm -f "$json"
}

# --- L2ToL2CrossDomainMessenger
run M1_window_lt $L2 's/if (_undeliveredAt <= sentAt + MESSAGE_EXPIRY_WINDOW)/if (_undeliveredAt < sentAt + MESSAGE_EXPIRY_WINDOW)/' \
  L2ToL2ExpiryHalmos "check_expire_iff check_expire_iff_unbounded check_expire_boundary"
run M2_relay_no07 $L2 's/if (target == Predeploys.L2_CROSS_DOMAIN_MESSENGER) revert MessageTargetL2CrossDomainMessenger();//' \
  L2ToL2ExpiryHalmos "check_UnsafeTargetRule_relay check_UnsafeTargetRule_relay_l2cdm check_OnlyExportReachesL1_relay_l2cdm"
run M3_send_no07 $L2 's/if (_target == Predeploys.L2_CROSS_DOMAIN_MESSENGER) revert MessageTargetL2CrossDomainMessenger();//' \
  L2ToL2ExpiryHalmos "check_UnsafeTargetRule_send"
run M3b_send_no23 $L2 's/if (_target == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert MessageTargetL2ToL2CrossDomainMessenger();//' \
  L2ToL2ExpiryHalmos "check_UnsafeTargetRule_send"
run M4_export_dest_is_source $L2 's/_destination: block.chainid,/_destination: _source,/' \
  L2ToL2ExpiryHalmos "check_export_binding"
run M5_export_no_relayed_check $L2 's/if (successfulMessages\[messageHash_\]) revert MessageAlreadyRelayed();//' \
  L2ToL2ExpiryHalmos "check_export_binding"
run M6_export_ts0 $L2 's/(messageHash_, block.timestamp)/(messageHash_, 0)/' \
  L2ToL2ExpiryHalmos "check_export_binding"
run M7_expire_no_xdomain_check $L2 's/|| ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).xDomainMessageSender()/|| address(0)/' \
  L2ToL2ExpiryHalmos "check_expire_iff check_expire_iff_unbounded"
run M13_send_timestamp_1 $L2 's/sentMessageTimestamps\[messageHash_\] = block.timestamp;/sentMessageTimestamps[messageHash_] = 1;/' \
  L2ToL2ExpiryHalmos "check_send_effects_and_frame"
run M14_relay_sets_expired $L2 's/        successfulMessages\[messageHash\] = true;/        successfulMessages[messageHash] = true; expiredMessages[messageHash] = true;/' \
  L2ToL2ExpiryHalmos "check_relay_effects_and_frame"
run M15_export_bumps_nonce $L2 's/if (successfulMessages\[messageHash_\]) revert MessageAlreadyRelayed();/if (successfulMessages[messageHash_]) revert MessageAlreadyRelayed(); msgNonce++;/' \
  L2ToL2ExpiryHalmos "check_export_binding"
run M16_send_no_sentMessages $L2 's/sentMessages\[nonce\] = messageHash_;//' \
  L2ToL2ExpiryHalmos "check_send_effects_and_frame"
run M17_expire_sets_successful $L2 's/expiredMessages\[_messageHash\] = true;/expiredMessages[_messageHash] = true; successfulMessages[_messageHash] = true;/' \
  L2ToL2ExpiryHalmos "check_expire_iff"
run M24_relay_no_mark_before_call $L2 's/        successfulMessages\[messageHash\] = true;//' \
  L2ToL2ExpiryHalmos "check_relay_effects_and_frame check_OnlyExportReachesL1_relay_reentrant"
# --- L1CrossDomainMessenger / CrossDomainMessenger
run M8_l1_no_lockbox $L1 's/|| !portal.ethLockbox().authorizedPortals(callerPortal)//' \
  L1CDMExpiryHalmos "check_relayUndelivered_iff_and_deposit"
run M9_l1_gas $L1 's/EXPIRE_MESSAGE_GAS_LIMIT = 100_000/EXPIRE_MESSAGE_GAS_LIMIT = 100_001/' \
  L1CDMExpiryHalmos "check_relayUndelivered_iff_and_deposit"
run M20_l1_self_not_unsafe $L1 's/return _target == address(this) || _target == address(portal);/return _target == address(portal);/' \
  L1CDMExpiryHalmos "check_L1_relayMessage_rejectsSelfAndPortalTargets"
run M21_cdm_sender_not_reset $CDM 's/^        xDomainMsgSender = Constants.DEFAULT_L2_SENDER;$//' \
  L1CDMExpiryHalmos "check_L1_xDomainMessageSender_isRelayedSender"
run M23_cdm_sender_is_origin $CDM 's/messageNonce(), msg.sender, _target, msg.value/messageNonce(), tx.origin, _target, msg.value/' \
  L1CDMExpiryHalmos "check_L1_sendMessage_senderFieldIsCaller check_relayUndelivered_iff_and_deposit"
# --- SuperchainETHBridge
run M10_refund_no_refunded_check $BR 's/if (refunded\[messageHash\]) revert AlreadyRefunded();//' \
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
