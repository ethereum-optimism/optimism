#!/usr/bin/env bash
# Keep the manual recipe unchanged; CI optionally retains complete Go reports.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
fork="${1:?Pass the pre-fork state name}"
export OP_E2E_GEN_PREFORK_STATE="$fork"
if [[ -n "${NUT_PREFORK_REPORT_ROOT:-}" ]]; then
  exec python3 ops/ci/runtime/nut-prefork.py --test-fork "$fork"
fi
exec go test -count=1 -run TestGenerateForkState ./rust/kona/tests/proofs/
