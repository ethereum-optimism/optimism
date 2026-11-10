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
# op-deployer never compiles, so warm forge's script cache and check that its scripts compile nothing.
FORGE="$forge" ./scripts/ops/warm-script-cache.sh
