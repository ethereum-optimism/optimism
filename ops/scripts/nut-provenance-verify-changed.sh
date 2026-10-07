#!/usr/bin/env bash
set -euo pipefail

# Verifies provenance for all forks whose bundle hash changed vs develop.
# For each fork whose hash changed, checks out the recorded commit,
# regenerates the bundle, and verifies it matches byte-for-byte.
# Unchanged forks are skipped to avoid expensive forge rebuilds.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"
args=(--provider circleci)
case "${CI_NUT_PROVENANCE_FULL:-false}" in
  true|1) args+=(--full) ;;
  false|0) ;;
  *) echo "CI_NUT_PROVENANCE_FULL must be true/false or 1/0" >&2; exit 1 ;;
esac
exec python3 ops/ci/runtime/nut-provenance.py "${args[@]}"
