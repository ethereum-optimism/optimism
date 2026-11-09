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
"$forge" build --skip "/**/test/**"
"$forge" build test/dispute/zk/MockSP1Verifier.sol
