#!/usr/bin/env bash
# Reusable compilation with isolated feature/profile tool caches.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
: "${CONTRACT_PREREQUISITES:?Contract preparation artifact must be mounted}"
mkdir -p .ci/contracts-prepare .ci/contracts-build
cp -a "${CONTRACT_PREREQUISITES}/." .ci/contracts-prepare/
tar -xzf .ci/contracts-prepare/source.tar.gz
cd packages/contracts-bedrock
# shellcheck disable=SC1091  # shared helper is checked separately and resolved at runtime
source ../../ops/ci/contracts-test-env.sh
# Native tool caches hold only compiler outputs. Inherited test outcomes and
# fuzz counterexamples cannot be used by either compilation or verdict tasks.
rm -rf results cache/test-failures cache/fuzz cache/invariant
mkdir -p results/reports
forge config --json >results/reports/foundry-config.json
python3 ../../ops/ci/contracts-test-report.py prepare results/reports/foundry-config.json
cp results/reports/foundry-config.json results/reports/test-files.txt ../../.ci/contracts-build/
forge build 2>&1 | tee ../../.ci/contracts-build/compiler.log
python3 ../../ops/ci/rwx-contracts-build.py package
