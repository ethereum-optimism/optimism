#!/usr/bin/env bash
set -euo pipefail
SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# Get the repo root (two levels up from ops/prestate-reproducibility/)
REPO_ROOT=$(cd "${SCRIPTS_DIR}/../.." && pwd)

TMP_DIR=$(mktemp -d)
WORKTREE_DIR="${TMP_DIR}/optimism"

function cleanup() {
  git -C "${REPO_ROOT}" worktree remove "${WORKTREE_DIR}" --force 2> /dev/null || true
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

echo "Creating worktree in: ${WORKTREE_DIR}"
# Create a detached worktree - we'll checkout specific tags in the build functions
git -C "${REPO_ROOT}" worktree add "${WORKTREE_DIR}" HEAD --detach

STATES_DIR="${SCRIPTS_DIR}/temp/states"
LOGS_DIR="${SCRIPTS_DIR}/temp/logs"
BIN_DIR="${WORKTREE_DIR}/op-program/bin/"
VERSIONS_FILE="${STATES_DIR}/versions.json"

mkdir -p "${STATES_DIR}" "${LOGS_DIR}"

cd "${WORKTREE_DIR}"

function build_prestates() {
  local version=$1
  local log_file=$2
  local short_version="${version#*/v}"
  echo "Building version: ${version} Logs: ${log_file}"

  git checkout --force "${version}" > "${log_file}" 2>&1

  if [ -f mise.toml ]; then
    echo "Install dependencies with mise" >> "${log_file}"
    # Install only the host-side tools needed to build prestates and extract hashes.
    mise trust
    mise install -v -y go just jq >> "${log_file}" 2>&1
  fi

  rm -rf "${BIN_DIR}"
  rm -rf rust/kona/prestate-artifacts-*
  if [ -f justfile ] && just --show reproducible-prestate &> /dev/null; then
    just reproducible-prestate >> "${log_file}" 2>&1
  else
    make reproducible-prestate >> "${log_file}" 2>&1
  fi

  if [[ "${version}" =~ ^kona-client/v ]]; then
    if [ -f "rust/kona/prestate-artifacts-cannon/prestate-proof.json" ]; then
      local hash
      hash=$(jq -r .pre rust/kona/prestate-artifacts-cannon/prestate-proof.json)
      cp rust/kona/prestate-artifacts-cannon/prestate.bin.gz "${STATES_DIR}/${hash}.bin.gz"
      VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${short_version}\", \"hash\": \"${hash}\", \"type\": \"cannon64-kona\"}]")
      echo "Built cannon64-kona ${version}: ${hash}"
    fi

    if [ -f "rust/kona/prestate-artifacts-cannon-interop/prestate-proof.json" ]; then
      local hash
      hash=$(jq -r .pre rust/kona/prestate-artifacts-cannon-interop/prestate-proof.json)
      cp rust/kona/prestate-artifacts-cannon-interop/prestate.bin.gz "${STATES_DIR}/${hash}.bin.gz"
      VERSIONS_JSON=$(echo "${VERSIONS_JSON}" | jq ". += [{\"version\": \"${short_version}\", \"hash\": \"${hash}\", \"type\": \"cannon64-kona-interop\"}]")
      echo "Built cannon64-kona-interop ${version}: ${hash}"
    fi
  fi
}

VERSIONS_JSON="[]"
readarray -t VERSIONS < <(git tag --list 'kona-client/v*' --sort=taggerdate)

for i in "${!VERSIONS[@]}"; do
  tag="${VERSIONS[i]}"
  log_file="${LOGS_DIR}/build-${tag//\//-}.txt"

  pushd .
  build_prestates "${tag}" "${log_file}"
  popd
  if [ "${CIRCLECI:-}" = "true" ]; then
    if (((i + 1) % 10 == 0)); then
      echo "Pruning docker build artifacts after ${i} builds"
      docker system prune -f
    fi
  fi
done

echo "${VERSIONS_JSON}" > "${VERSIONS_FILE}"
echo "All prestates successfully built and available in ${STATES_DIR}"
