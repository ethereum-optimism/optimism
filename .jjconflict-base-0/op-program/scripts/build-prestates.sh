#!/usr/bin/env bash
set -euo pipefail
SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
KONA_REPO_URL=https://github.com/op-rs/kona

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

function build_kona_prestate() {
  local version=$1
  if [[ -z "${version}" ]]; then
    echo "Error: version is required"
    exit 1
  fi
  local short_version
  short_version=$(echo "${version}" | cut -c 14-)
  local log_file="${LOGS_DIR}/build-kona-${short_version}.txt"
  echo "Building Version: ${version} Logs: ${log_file}"

  mkdir -p kona-prestate-build
  cd kona-prestate-build

  if [[ -d kona ]]; then
    cd kona
    git checkout --force "${version}" > "${log_file}" 2>&1
  else
    git clone -b "${version}" "$KONA_REPO_URL" kona > "${log_file}" 2>&1
    cd kona
  fi
  # kona doesn't define a just dependency in its mise config.
  # but the monorepo does and it should be preinstalled by now. So let's setup the just shim.
  MISE_DEFAULT_CONFIG_FILENAME="${REPO_DIR}"/mise.toml mise use just > "${log_file}" 2>&1

  cd docker/fpvm-prestates
  rm -rf ../../prestate-artifacts-cannon
  just cannon kona-client "${version}" "$(cat ../../.config/cannon_tag)" >> "${log_file}" 2>&1
  local prestate_hash
  prestate_hash=$(cat ../../prestate-artifacts-cannon/prestate-proof.json | jq -r .pre)
  cp ../../prestate-artifacts-cannon/prestate.bin.gz "${STATES_DIR}/${prestate_hash}.bin.gz"
  VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${short_version}\", \"hash\": \"${prestate_hash}\", \"type\": \"cannon64-kona\"}]")
  echo "Built kona ${version}: ${prestate_hash}"

  rm ../../prestate-artifacts-cannon/prestate-proof.json
  just cannon kona-client-int "${version}" "$(cat ../../.config/cannon_tag)" >> "${log_file}" 2>&1
  prestate_hash=$(cat ../../prestate-artifacts-cannon/prestate-proof.json | jq -r .pre)
  cp ../../prestate-artifacts-cannon/prestate.bin.gz "${STATES_DIR}/${prestate_hash}.bin.gz"
  VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${short_version}\", \"hash\": \"${prestate_hash}\", \"type\": \"cannon64-kona-interop\"}]")
  echo "Built kona-interop ${version}: ${prestate_hash}"
}

function build_op_program_prestate() {
  local VERSION=$1
  if [[ -z "${VERSION}" ]]; then
    echo "Error: VERSION is required"
    exit 1
  fi
  local SHORT_VERSION # declared separately from assignment to avoid masking failures
  SHORT_VERSION=$(echo "${VERSION}" | cut -c 13-)
  local LOG_FILE="${LOGS_DIR}/build-${SHORT_VERSION}.txt"
  echo "Building Version: ${VERSION} Logs: ${LOG_FILE}"
  # use --force to overwrite any mise.toml changes
  git checkout --force "${VERSION}" > "${LOG_FILE}" 2>&1
  if [ -f mise.toml ]; then
    echo "Install dependencies with mise" >> "${LOG_FILE}"
    # we rely only on go and jq for the reproducible-prestate build.
    # The mise cache should already have jq preinstalled
    # But we need to ensure that this ${VERSION} has the correct go version
    # So we replace the mise.toml with a minimal one that only specifies go
    # Otherwise, `mise install` fails as it conflicts with other preinstalled dependencies
    GO_VERSION=$(mise config get tools.go)
    cat > mise.toml << EOF
[tools]
go = "${GO_VERSION}"
EOF
    mise install -v -y >> "${LOG_FILE}" 2>&1
  fi
  rm -rf "${BIN_DIR}"
  make reproducible-prestate >> "${LOG_FILE}" 2>&1

  if [ -f "${BIN_DIR}/prestate-proof.json" ]; then
    local HASH
    HASH=$(cat "${BIN_DIR}/prestate-proof.json" | jq -r .pre)
    if [ -f "${BIN_DIR}/prestate.bin.gz" ]; then
      cp "${BIN_DIR}/prestate.bin.gz" "${STATES_DIR}/${HASH}.bin.gz"
    else
      cp "${BIN_DIR}/prestate.json" "${STATES_DIR}/${HASH}.json"
    fi
    VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${SHORT_VERSION}\", \"hash\": \"${HASH}\", \"type\": \"cannon32\"}]")
    echo "Built cannon32 ${VERSION}: ${HASH}"
  fi

  if [ -f "${BIN_DIR}/prestate-proof-mt64.json" ]; then
    local HASH
    HASH=$(cat "${BIN_DIR}/prestate-proof-mt64.json" | jq -r .pre)
    cp "${BIN_DIR}/prestate-mt64.bin.gz" "${STATES_DIR}/${HASH}.mt64.bin.gz"
    VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${SHORT_VERSION}\", \"hash\": \"${HASH}\", \"type\": \"cannon64\"}]")
    echo "Built cannon64 ${VERSION}: ${HASH}"
  fi

  if [ -f "${BIN_DIR}/prestate-proof-interop.json" ]; then
    local HASH
    HASH=$(cat "${BIN_DIR}/prestate-proof-interop.json" | jq -r .pre)
    cp "${BIN_DIR}/prestate-interop.bin.gz" "${STATES_DIR}/${HASH}.interop.bin.gz"
    VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${SHORT_VERSION}\", \"hash\": \"${HASH}\", \"type\": \"interop\"}]")
    echo "Built cannon-interop ${VERSION}: ${HASH}"
  fi
}

# this global is written to by build_op_program_prestate and build_kona_prestate
VERSIONS_JSON="[]"
readarray -t VERSIONS < <(git tag --list 'op-program/v*' --sort taggerdate)

for i in "${!VERSIONS[@]}"; do
  pushd .
  build_op_program_prestate "${VERSIONS[i]}"
  popd
  # after every 10 builds, cleanup docker artifacts to reclaim disk space
  if [ "$CIRCLECI" = "true" ]; then
    if (( (i + 1) % 10 == 0 )); then
      echo "Pruning docker build artifacts after ${i} builds"
      docker system prune -f
    fi
  fi
done
echo "${VERSIONS_JSON}" > "${VERSIONS_FILE}"

# ignore alpha, beta, and older kona-client releases. The cannon prestate builds are not well supported on these versions
EXCLUDED=(
  "kona-client/v1.0.0"
  "kona-client/v1.0.1"
  "kona-client/v1.0.2"
  "kona-client/v1.1.0-rc.1"
  "kona-client/v1.1.0-rc.3"
  "kona-client/v1.1.3"
)
printf "%s\n" "${EXCLUDED[@]}" > excluded.txt

readarray -t KONA_VERSIONS < <(git ls-remote --tags "$KONA_REPO_URL" | grep kona-client/ \
  | sed 's|.*refs/tags/||' | sed 's/\^{}//' | sort -u \
  | grep -v beta | grep -v alpha | grep -v -F -f excluded.txt)

for i in "${!KONA_VERSIONS[@]}"; do
  pushd .
  build_kona_prestate "${KONA_VERSIONS[i]}"
  popd
  # after every 10 builds, cleanup docker artifacts to reclaim disk space
  if [ "$CIRCLECI" = "true" ]; then
    if (( (i + 1) % 10 == 0 )); then
      echo "Pruning docker build artifacts after ${i} builds"
      docker system prune -f
    fi
  fi
done
echo "${VERSIONS_JSON}" > "${VERSIONS_FILE}"

echo "All prestates successfully built and available in ${STATES_DIR}"
