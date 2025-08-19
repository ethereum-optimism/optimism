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
VERSIONS_FILE="${STATES_DIR}/versions.json"

mkdir -p "${STATES_DIR}" "${LOGS_DIR}"


cd "${REPO_DIR}"

VERSIONS_JSON="[]"
VERSIONS=$(git tag --list 'op-program/v*' --sort taggerdate | grep v1.4.0-rc.3)
#VERSIONS=$(git tag --list 'op-program/v*' --sort taggerdate | grep v1.6.0-rc.2)

for VERSION in ${VERSIONS}
do
    SHORT_VERSION=$(echo "${VERSION}" | cut -c 13-)
    LOG_FILE="${LOGS_DIR}/build-${SHORT_VERSION}.txt"
    echo "Building Version: ${VERSION} Logs: ${LOG_FILE}"
    git checkout "${VERSION}" > "${LOG_FILE}" 2>&1
    if [ -f mise.toml ]
    then
      echo "Install dependencies with mise" 2>&1 | tee "${LOG_FILE}"
      #rustup default stable
      #mise install go -v -y 2>&1 | tee "${LOG_FILE}"
      # install only the tools used by the reproducible-build; go and jq.
      GO_VERSION=$(cat mise.toml | grep -E '^go\s+=\s+"[0-9]+.*"$' | sed 's/go = "\(.*\)"/\1/')
      JQ_VERSION=$(cat mise.toml | grep -E '^jq\s+=\s+"[0-9]+.*"$' | sed 's/jq = "\(.*\)"/\1/')
      if [ -z "${GO_VERSION}" ] || [ -z "${JQ_VERSION}" ]; then
        echo "Error: go or jq version not found in mise.toml for the ${VERSION} release"
        exit 1
      fi
      echo "installing go@${GO_VERSION} and jq@${JQ_VERSION}"
      export MISE_NO_CONFIG=1
      mise install "go@${GO_VERSION}" "jq@${JQ_VERSION}" -v -y 2>&1 | tee "${LOG_FILE}"
      echo "done installing deps"
      if [ ! -x "$(command -v jq)" ]; then
        echo "debugme: jq is not installed!"
        exit 1
      fi
      echo "found jq"
      which jq
      echo "found which jq"
      mise use -g "go@${GO_VERSION}"
      echo "using go"
      mise use -g "jq@${JQ_VERSION}"
      echo "using jq"
      mise reshim
      echo "reshim done"
      mise which jq
      echo "which jq done"
      echo "jq version is $(jq --version)"
    fi
    rm -rf "${BIN_DIR}"
    echo "building prestate"
    make reproducible-prestate 2>&1 | tee "${LOG_FILE}"

    if [ -f "${BIN_DIR}/prestate-proof.json" ]; then
      HASH=$(cat "${BIN_DIR}/prestate-proof.json" | jq -r .pre)
      if [ -f "${BIN_DIR}/prestate.bin.gz" ]
      then
        cp "${BIN_DIR}/prestate.bin.gz" "${STATES_DIR}/${HASH}.bin.gz"
      else
        cp "${BIN_DIR}/prestate.json" "${STATES_DIR}/${HASH}.json"
      fi
      VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${SHORT_VERSION}\", \"hash\": \"${HASH}\", \"type\": \"cannon32\"}]")
      echo "Built cannon32 ${VERSION}: ${HASH}"
    fi

    if [ -f "${BIN_DIR}/prestate-proof-mt64.json" ]; then
      HASH=$(cat "${BIN_DIR}/prestate-proof-mt64.json" | jq -r .pre)
      cp "${BIN_DIR}/prestate-mt64.bin.gz" "${STATES_DIR}/${HASH}.mt64.bin.gz"
      VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${SHORT_VERSION}\", \"hash\": \"${HASH}\", \"type\": \"cannon64\"}]")
      echo "Built cannon64 ${VERSION}: ${HASH}"
    fi

    if [ -f "${BIN_DIR}/prestate-proof-interop.json" ]; then
      HASH=$(cat "${BIN_DIR}/prestate-proof-interop.json" | jq -r .pre)
      cp "${BIN_DIR}/prestate-interop.bin.gz" "${STATES_DIR}/${HASH}.interop.bin.gz"
      VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${SHORT_VERSION}\", \"hash\": \"${HASH}\", \"type\": \"interop\"}]")
      echo "Built cannon-interop ${VERSION}: ${HASH}"
    fi
done
echo "${VERSIONS_JSON}" > "${VERSIONS_FILE}"

echo "All prestates successfully built and available in ${STATES_DIR}"
