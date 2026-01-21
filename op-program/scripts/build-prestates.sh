#!/usr/bin/env bash
set -euo pipefail
SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# Get the repo root (two levels up from op-program/scripts/)
REPO_ROOT=$(cd "${SCRIPTS_DIR}/../.." && pwd)

TMP_DIR=$(mktemp -d)
WORKTREE_DIR="${TMP_DIR}/optimism"

function cleanup() {
  git -C "${REPO_ROOT}" worktree remove "${WORKTREE_DIR}" --force 2>/dev/null || true
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

echo "Creating worktree in: ${WORKTREE_DIR}"
# Create a detached worktree - we'll checkout specific tags in the build functions
git -C "${REPO_ROOT}" worktree add "${WORKTREE_DIR}" HEAD --detach

STATES_DIR="${SCRIPTS_DIR}/../temp/states"
LOGS_DIR="${SCRIPTS_DIR}/../temp/logs"
BIN_DIR="${WORKTREE_DIR}/op-program/bin/"
VERSIONS_FILE="${STATES_DIR}/versions.json"

mkdir -p "${STATES_DIR}" "${LOGS_DIR}"

cd "${WORKTREE_DIR}"

# Legacy kona versions that were tagged in the old op-rs/kona repo before migration
LEGACY_KONA_VERSIONS=(
  "kona-client/v1.1.6"
  "kona-client/v1.1.7"
  "kona-client/v1.2.0"
  "kona-client/v1.2.1"
  "kona-client/v1.2.2"
  "kona-client/v1.2.4"
  "kona-client/v1.2.5"
  "kona-client/v1.2.7"
)
LEGACY_KONA_REPO="https://github.com/op-rs/kona"
LEGACY_KONA_DIR="${TMP_DIR}/kona-legacy"

function build_legacy_kona_prestate() {
  local version=$1
  local short_version=$2
  local log_file=$3
  echo "Building Version: ${version} Logs: ${log_file}"

  mkdir -p "${LEGACY_KONA_DIR}"
  cd "${LEGACY_KONA_DIR}"

  if [[ -d kona ]]; then
    cd kona
    git checkout --force "${version}" > "${log_file}" 2>&1
  else
    git clone -b "${version}" "${LEGACY_KONA_REPO}" kona > "${log_file}" 2>&1
    cd kona
  fi
  # kona doesn't define a just dependency in its mise config.
  # but the monorepo does and it should be preinstalled by now. So let's setup the just shim.
  MISE_DEFAULT_CONFIG_FILENAME="${WORKTREE_DIR}"/mise.toml mise use just > "${log_file}" 2>&1

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

  cd "${WORKTREE_DIR}"
}

function build_kona_prestate() {
  local version=$1
  local short_version=$2
  local log_file=$3
  echo "Building kona version: ${version} Logs: ${log_file}"

  git checkout --force "${version}" > "${log_file}" 2>&1

  # Clean previous build artifacts
  rm -rf kona/prestate-artifacts-*

  # Build kona prestates using the standard build target
  (cd kona && just reproducible-prestate) >> "${log_file}" 2>&1

  # Collect cannon prestate
  local prestate_hash
  prestate_hash=$(jq -r .pre kona/prestate-artifacts-cannon/prestate-proof.json)
  cp kona/prestate-artifacts-cannon/prestate.bin.gz "${STATES_DIR}/${prestate_hash}.bin.gz"
  VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${short_version}\", \"hash\": \"${prestate_hash}\", \"type\": \"cannon64-kona\"}]")
  echo "Built cannon64-kona ${version}: ${prestate_hash}"

  # Collect interop prestate
  prestate_hash=$(jq -r .pre kona/prestate-artifacts-cannon-interop/prestate-proof.json)
  cp kona/prestate-artifacts-cannon-interop/prestate.bin.gz "${STATES_DIR}/${prestate_hash}.bin.gz"
  VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${short_version}\", \"hash\": \"${prestate_hash}\", \"type\": \"cannon64-kona-interop\"}]")
  echo "Built cannon64-kona-interop ${version}: ${prestate_hash}"
}

function build_op_program_prestate() {
  local VERSION=$1
  local SHORT_VERSION=$2
  local LOG_FILE=$3
  echo "Building op-program version: ${VERSION} Logs: ${LOG_FILE}"
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
  make -C op-program reproducible-prestate >> "${LOG_FILE}" 2>&1

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
  tag="${VERSIONS[i]}"
  # e.g. op-program/v1.2.3 -> v1.2.3
  short_version="${tag#*/}"
  log_file="${LOGS_DIR}/build-op-program-${short_version}.txt"

  pushd .
  build_op_program_prestate "${tag}" "${short_version}" "${log_file}"
  popd
  # after every 10 builds, cleanup docker artifacts to reclaim disk space
  if [ "${CIRCLECI:-}" = "true" ]; then
    if (( (i + 1) % 10 == 0 )); then
      echo "Pruning docker build artifacts after ${i} builds"
      docker system prune -f
    fi
  fi
done

# Build legacy kona prestates from the old op-rs/kona repo
for i in "${!LEGACY_KONA_VERSIONS[@]}"; do
  tag="${LEGACY_KONA_VERSIONS[i]}"
  short_version="${tag#*/}"
  log_file="${LOGS_DIR}/build-kona-${short_version}.txt"

  pushd .
  build_legacy_kona_prestate "${tag}" "${short_version}" "${log_file}"
  popd
  if [ "${CIRCLECI:-}" = "true" ]; then
    if (( (i + 1) % 10 == 0 )); then
      echo "Pruning docker build artifacts after ${i} builds"
      docker system prune -f
    fi
  fi
done

# Build kona prestates from tags in this monorepo (versions after migration)
readarray -t KONA_VERSIONS < <(git tag --list 'kona-client/v*' --sort taggerdate)

for i in "${!KONA_VERSIONS[@]}"; do
  tag="${KONA_VERSIONS[i]}"
  # e.g. kona-client/v1.2.3 -> v1.2.3
  short_version="${tag#*/}"
  log_file="${LOGS_DIR}/build-kona-${short_version}.txt"

  pushd .
  build_kona_prestate "${tag}" "${short_version}" "${log_file}"
  popd
  # after every 10 builds, cleanup docker artifacts to reclaim disk space
  if [ "${CIRCLECI:-}" = "true" ]; then
    if (( (i + 1) % 10 == 0 )); then
      echo "Pruning docker build artifacts after ${i} builds"
      docker system prune -f
    fi
  fi
done

echo "${VERSIONS_JSON}" > "${VERSIONS_FILE}"
echo "All prestates successfully built and available in ${STATES_DIR}"
