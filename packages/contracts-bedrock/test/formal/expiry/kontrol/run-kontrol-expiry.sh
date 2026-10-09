#!/bin/bash
# Kontrol proofs for per-message interop expiry (see the .k.sol files under solc0815/ and solc0825/,
# grouped by the compiler version of the contract under test).
#
# Requires kontrol 1.0.255 on PATH (the version pinned in mise.toml), e.g. inside the image
# runtimeverificationinc/kontrol:ubuntu-jammy-1.0.255, and the `kexpiry` foundry profile:
#
#   [profile.kexpiry]
#   src = "test/formal/expiry/kontrol"
#   out = "test/formal/expiry/kontrol/out"
#   test = "test/formal/expiry/kontrol"
#   script = "test/formal/expiry/kontrol"
#
# Usage (from packages/contracts-bedrock):
#   test/formal/expiry/kontrol/run-kontrol-expiry.sh                 # build + every proof
#   test/formal/expiry/kontrol/run-kontrol-expiry.sh <Contract.test>... # build + only those
# Env: KONTROL_WORKERS (default 16), KONTROL_NO_BUILD=1 to skip the build, KONTROL_FRESH=1 to discard
#      saved proof state (default: resume unfinished proofs; changed code is re-initialised by kontrol).
#
# Expected results: every prove_* passes EXCEPT *_WITNESS (non-vacuity witnesses, expected to FAIL with a
# counterexample). witnesses.tsv pairs each proof with the witness that shares its assumptions, and
# check-results.py (run at the end) fails the script unless every proof passed and every witness failed.
# When only some tests are selected, the check reports the unselected ones as "no result".
set -Eeuo pipefail

export FOUNDRY_PROFILE=kexpiry
cd "$(dirname "${BASH_SOURCE[0]}")/../../../.."

if [ "${KONTROL_NO_BUILD:-0}" != 1 ]; then
  lemmas=test/formal/expiry/kontrol/expiry-lemmas.md
  # kontrol keeps its first copy of a required file; drop it so lemma edits are picked up.
  rm -rf test/formal/expiry/kontrol/out/kompiled/requires
  kontrol build --no-metadata --rekompile --regen \
    --require "$lemmas" \
    --module-import L2ToL2CrossDomainMessengerExpiryKontrol:EXPIRY-LEMMAS \
    --module-import L1CrossDomainMessengerExpiryKontrol:EXPIRY-LEMMAS \
    --module-import SuperchainETHBridgeExpiryKontrol:EXPIRY-LEMMAS \
    --module-import UndeliveredMessageExporterExpiryKontrol:EXPIRY-LEMMAS
fi

fresh=()
if [ "${KONTROL_FRESH:-0}" = 1 ]; then fresh=(--remove-old-proofs); fi

tests=()
for t in "$@"; do tests+=(--match-test "$t"); done

kontrol prove \
  "${tests[@]}" \
  --max-depth 10000 \
  --max-iterations 10000 \
  --workers "${KONTROL_WORKERS:-16}" \
  --kore-rpc-command 'kore-rpc-booster --no-post-exec-simplify --equation-max-recursion 100 --equation-max-iterations 1000' \
  --xml-test-report \
  --maintenance-rate 16 \
  --assume-defined \
  --no-log-rewrites \
  --smt-timeout 16000 \
  --smt-retry-limit 0 \
  --no-stack-checks \
  "${fresh[@]}" || true

# Non-vacuity gate: every proof must pass and every paired witness must fail with a counterexample.
results=$(mktemp)
kontrol list > "$results"
python3 test/formal/expiry/kontrol/check-results.py "$results"
