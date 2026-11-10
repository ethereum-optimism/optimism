#!/usr/bin/env bash
set -euo pipefail

# Compiles the contracts that op-deployer embeds: contracts-bedrock without its tests, plus the
# MockSP1Verifier that `apply --deploy-mock-sp1-verifier` deploys. Every op-deployer build (justfile,
# Docker and melange images, release binaries) compiles through this script so they all embed the
# same artifacts.
#
# Usage: build-embedded-contracts.sh. Set FORGE to use a forge other than the one on PATH.

forge="${FORGE:-forge}"
cd "$(dirname "${BASH_SOURCE[0]}")/../../packages/contracts-bedrock"
# Same as contracts-bedrock's `build-no-tests` recipe; the Docker contracts stage has no just.
# `--deny never` keeps forge's compile cache in the artifacts, so op-deployer doesn't recompile.
"$forge" build --deny never --skip "/**/test/**"
"$forge" build --deny never test/dispute/zk/MockSP1Verifier.sol
# The skip leaves out the test/mocks that the scripts import, so compile the scripts too.
"$forge" build --deny never scripts
# `forge script` compiles its own source set, which needs artifacts `forge build` doesn't produce.
# A compile-only run puts them in the cache, as contracts-bedrock's `forge-build` recipe does, so
# op-deployer's `forge script` runs don't recompile. It fails because the signature doesn't exist.
out=$("$forge" script scripts/deploy/Deploy.s.sol --deny never --skip "/**/test/**" \
  --sig "idonotexist()" --skip-simulation 2>&1) || true
if ! grep -q 'is not implemented in your script' <<< "$out"; then
  echo "$out" >&2
  exit 1
fi
# op-deployer never compiles, so check that a second run compiles nothing, as `forge-build` does.
# Check each script that op-deployer can run, the ones with runWithBytes: `forge script` checks the
# cache of src/ and of its target's imports, not of the other scripts. No --skip, as op-deployer runs
# them: a script can import from test/.
scripts=$(find scripts -name '*.s.sol' -exec grep -l 'function runWithBytes' {} +) || true
if [ -z "$scripts" ]; then
  echo "found no op-deployer scripts" >&2
  exit 1
fi
for script in $scripts; do
  out=$("$forge" script "$script" --deny never --sig "idonotexist()" --skip-simulation 2>&1) || true
  if ! grep -q 'No files changed, compilation skipped' <<< "$out"; then
    echo "$out" >&2
    echo "the forge script cache of $script is not warm" >&2
    exit 1
  fi
done
