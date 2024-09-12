#!/bin/bash
set -euo pipefail
SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

TMP_DIR=$(mktemp -d)
function cleanup() {
    rm -rf "${TMP_DIR}"
}
trap cleanup EXIT
echo "Using temp dir: ${TMP_DIR}"
cd "${TMP_DIR}"

# Need to check out a fresh copy of the monorepo so we can switch to specific tags without it also affecting the
# contents of this script (which is checked into the repo).
git clone https://github.com/ethereum-optimism/optimism --recurse-submodules

STATES_DIR="${SCRIPTS_DIR}/../temp/states"
LOGS_DIR="${SCRIPTS_DIR}/../temp/logs"
REPO_DIR="${TMP_DIR}/optimism"
BIN_DIR="${REPO_DIR}/op-program/bin/"

mkdir -p "${STATES_DIR}" "${LOGS_DIR}"

cd "${REPO_DIR}"

VERSIONS=$(git tag | grep 'op-program\/v')

for VERSION in ${VERSIONS}
do
    LOG_FILE="${LOGS_DIR}/build-$(echo "${VERSION}" | cut -c 12-).txt"
    echo "Building Version: ${VERSION} Logs: ${LOG_FILE}"
    git checkout "${VERSION}" > "${LOG_FILE}" 2>&1
    make reproducible-prestate >> "${LOG_FILE}" 2>&1
    HASH=$(cat "${BIN_DIR}/prestate-proof.json" | jq -r .pre)
    cp "${BIN_DIR}/prestate.json" "${STATES_DIR}/${HASH}.json"
    echo "Built ${VERSION}: ${HASH}"
done

echo "All prestates successfully built and available in ${STATES_DIR}"
