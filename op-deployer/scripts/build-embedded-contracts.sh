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
"$forge" build --skip "/**/test/**"
"$forge" build test/dispute/zk/MockSP1Verifier.sol
# `forge script` compiles its own source set, which needs artifacts `forge build` doesn't produce.
# A compile-only run puts them in the cache, as contracts-bedrock's `forge-build` recipe does, so
# op-deployer's `forge script` runs don't recompile. It fails because the signature doesn't exist.
out=$("$forge" script scripts/deploy/Deploy.s.sol --skip "/**/test/**" --sig "idonotexist()" \
  --skip-simulation 2>&1) || true
if ! grep -q 'is not implemented in your script' <<<"$out"; then
  echo "$out" >&2
  exit 1
fi
