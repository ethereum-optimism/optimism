#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

if [[ $# -lt 1 ]]; then
  echo "usage: $0 CONTRACT_NAME" >&2
  exit 1
fi

CONTRACT="$1"
ARTIFACT_DIR="packages/contracts-bedrock/forge-artifacts/${CONTRACT}.sol"
ARTIFACT_PATH="${ARTIFACT_DIR}/${CONTRACT}.json"

if [[ ! -f "${ARTIFACT_PATH}" ]]; then
  echo "error: artifact not found at ${ARTIFACT_PATH}. Run the contracts build first." >&2
  exit 1
fi

OUTPUT_BASENAME="$(echo "${CONTRACT}" | tr '[:upper:]' '[:lower:]')"
OUTPUT_PATH="op-e2e/bindings/${OUTPUT_BASENAME}.go"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

ABI_PATH="${TMPDIR}/${CONTRACT}.abi.json"
BIN_PATH="${TMPDIR}/${CONTRACT}.bin"

jq '.abi' "${ARTIFACT_PATH}" > "${ABI_PATH}"
jq -r '.bytecode.object' "${ARTIFACT_PATH}" > "${BIN_PATH}"

abigen --pkg bindings --type "${CONTRACT}" --abi "${ABI_PATH}" --bin "${BIN_PATH}" --out "${OUTPUT_PATH}"
gofmt -w "${OUTPUT_PATH}"
