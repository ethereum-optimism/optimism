#!/usr/bin/env bash
set -euo pipefail
SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# Get the repo root (two levels up from ops/prestate-reproducibility/)
REPO_ROOT=$(cd "${SCRIPTS_DIR}/../.." && pwd)

TMP_DIR=$(mktemp -d)
WORKTREE_DIR="${TMP_DIR}/optimism"

function cleanup() {
  local docker_target="${WORKTREE_DIR}/rust/kona/sp1/programs/target/elf-compilation/docker"
  if [[ -d "$docker_target" ]]; then
    local worktree_real docker_parent
    if worktree_real=$(cd -P "$WORKTREE_DIR" && pwd) &&
      docker_parent=$(cd -P "$(dirname "$docker_target")" && pwd) &&
      [[ "$docker_parent" == "$worktree_real"/* ]]; then
      if ! rm -rf "$docker_target" 2>/dev/null && ! sudo -n rm -rf "$docker_target"; then
        echo "warning: could not remove ${docker_target}" >&2
      fi
    else
      echo "warning: refusing to remove ${docker_target} outside the worktree" >&2
    fi
  fi
  git -C "${REPO_ROOT}" worktree remove "${WORKTREE_DIR}" --force 2> /dev/null || true
  rm -rf "${TMP_DIR}" || echo "warning: could not remove ${TMP_DIR}" >&2
}
trap cleanup EXIT

STATES_DIR="${SCRIPTS_DIR}/temp/states"
LOGS_DIR="${SCRIPTS_DIR}/temp/logs"
BIN_DIR="${WORKTREE_DIR}/op-program/bin/"
VERSIONS_FILE="${STATES_DIR}/versions.json"
STANDARD_PRESTATES_SNAPSHOT_FILE="${STATES_DIR}/standard-prestates.toml"
KONA_SP1_VERSIONS_FILE="${TMP_DIR}/kona-sp1-versions.txt"

mkdir -p "${STATES_DIR}" "${LOGS_DIR}"
(cd "${REPO_ROOT}" && mise exec -- go run ./ops/prestate-reproducibility/prestates/kona-sp1-versions --output "${STANDARD_PRESTATES_SNAPSHOT_FILE}") > "${KONA_SP1_VERSIONS_FILE}"

echo "Creating worktree in: ${WORKTREE_DIR}"
# Create a detached worktree - we'll checkout specific tags in the build functions
git -C "${REPO_ROOT}" worktree add "${WORKTREE_DIR}" HEAD --detach
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

function fail_kona_sp1() {
  echo "Kona SP1 version $1: $2" >&2
  return 1
}

function build_kona_sp1() {
  local version=$1
  local log_file=$2
  local ref="refs/tags/kona-sp1-program/v${version}"
  local commit
  commit=$(git rev-parse --verify "${ref}^{commit}" 2>/dev/null) || {
    fail_kona_sp1 "$version" "missing tag ${ref}"
    return 1
  }
  echo "Kona SP1 version ${version}: ${ref} -> ${commit}; logs: ${log_file}"
  git checkout --detach --force "$commit" > "$log_file" 2>&1 || {
    fail_kona_sp1 "$version" "failed to check out ${commit}"
    return 1
  }
  git clean -fdX -- rust/kona/sp1/elf >> "$log_file" 2>&1
  if [[ -n "${KONA_CUSTOM_CONFIGS_DIR:-}" ]]; then
    fail_kona_sp1 "$version" "KONA_CUSTOM_CONFIGS_DIR must be unset"
    return 1
  fi
  if [[ ! -f rust/kona/sp1/justfile ]]; then
    fail_kona_sp1 "$version" "tag has no SP1 guest recipe"
    return 1
  fi
  if ! (mise trust && mise install -v -y go just jq 'github:succinctlabs/sp1') 2>&1 | tee -a "$log_file"; then
    fail_kona_sp1 "$version" "failed to install tag-pinned tools"
    return 1
  fi
  local metadata unsafe_config_fallback
  metadata=$(mise exec -- cargo metadata --manifest-path rust/kona/sp1/programs/Cargo.toml --locked --format-version 1 2>> "$log_file") || {
    fail_kona_sp1 "$version" "failed to resolve guest features"
    return 1
  }
  unsafe_config_fallback=$(printf '%s\n' "$metadata" | jq -r '
    [.packages[] | select(.name == "kona-sp1-super-range") | .id] as $guests |
    any(.resolve.nodes[];
      (.id as $id | $guests | index($id)) != null and
      (.features | index("test-config-fallback")) != null)
  ') || {
    fail_kona_sp1 "$version" "failed to inspect guest features"
    return 1
  }
  if [[ "$unsafe_config_fallback" != "false" ]]; then
    fail_kona_sp1 "$version" "test-config-fallback must be disabled for production prestates"
    return 1
  fi
  if ! (cd rust/kona/sp1 && mise exec -- just build-elfs) 2>&1 | tee -a "$log_file"; then
    fail_kona_sp1 "$version" "tagged Docker build failed"
    return 1
  fi

  local manifest="rust/kona/sp1/elf/vkeys.toml"
  local elf="rust/kona/sp1/elf/super-aggregation-elf"
  local range_elf="rust/kona/sp1/elf/super-range-elf"
  local line hash derived output
  [[ -f "$manifest" && -s "$elf" && -s "$range_elf" ]] || {
    fail_kona_sp1 "$version" "missing guest manifest or ELF"
    return 1
  }
  local marker_status=0
  grep -aqF 'KONA_SP1_UNSAFE_TEST_CONFIG_FALLBACK{fd6d88e711058eef5eff1512237c8ad3}' "$range_elf" || marker_status=$?
  if [[ "$marker_status" -eq 0 ]]; then
    fail_kona_sp1 "$version" "test-config-fallback must be disabled for production prestates (unsafe ELF marker)"
    return 1
  elif [[ "$marker_status" -ne 1 ]]; then
    fail_kona_sp1 "$version" "could not scan super-range ELF for test-config-fallback"
    return 1
  fi
  if grep -Eq '^[[:space:]]*git_sha[[:space:]]*=[[:space:]]*"[^"]*-test"[[:space:]]*$' "$manifest"; then
    fail_kona_sp1 "$version" "test-config-fallback must be disabled for production prestates (test build marker)"
    return 1
  fi
  line=$(grep -E '^[[:space:]]*super-aggregation[[:space:]]*=' "$manifest" || true)
  if [[ ! "$line" =~ ^[[:space:]]*super-aggregation[[:space:]]*=[[:space:]]*\"(0x[0-9a-f]{64})\"[[:space:]]*$ ]]; then
    fail_kona_sp1 "$version" "missing, duplicate, or malformed super-aggregation vkey"
    return 1
  fi
  hash="${BASH_REMATCH[1]}"
  if [[ "$hash" =~ ^0x0{64}$ ]]; then
    fail_kona_sp1 "$version" "zero super-aggregation vkey"
    return 1
  fi
  output=$(SP1_PROVER=cpu mise exec -- cargo prove vkey --elf "$elf" 2>&1) || {
    fail_kona_sp1 "$version" "could not derive vkey from aggregation ELF: ${output}"
    return 1
  }
  derived=$(printf '%s\n' "$output" | grep -oE '0x[[:alnum:]]+' || true)
  if [[ ! "$derived" =~ ^0x[0-9a-fA-F]{64}$ ]]; then
    fail_kona_sp1 "$version" "ELF produced no unique bytes32 vkey"
    return 1
  fi
  if [[ "$(printf '%s' "$hash" | tr 'A-F' 'a-f')" != "$(printf '%s' "$derived" | tr 'A-F' 'a-f')" ]]; then
    fail_kona_sp1 "$version" "manifest vkey ${hash} differs from ELF vkey ${derived}"
    return 1
  fi
  VERSIONS_JSON=$(printf '%s\n' "$VERSIONS_JSON" | jq --arg version "$version" --arg hash "$hash" '. + [{version: $version, hash: $hash, type: "kona-sp1"}]')
  echo "Kona SP1 version ${version}: rebuilt super-aggregation vkey ${hash}"
}

VERSIONS_JSON="[]"
git tag --list 'kona-client/v*' --sort=taggerdate > "${TMP_DIR}/cannon-tags"
VERSIONS=()
while IFS= read -r tag; do
  VERSIONS+=("$tag")
done < "${TMP_DIR}/cannon-tags"

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

kona_count=0
while IFS= read -r -u 3 version; do
  [[ -n "$version" ]] || continue
  kona_count=$((kona_count + 1))
  log_file="${LOGS_DIR}/build-kona-sp1-program-v${version}.txt"
  build_kona_sp1 "$version" "$log_file"
done 3< "$KONA_SP1_VERSIONS_FILE"
if [[ "$kona_count" -eq 0 ]]; then
  echo "Kona SP1 registry selection: 0 entries; Cannon checks completed"
fi

echo "${VERSIONS_JSON}" > "${VERSIONS_FILE}"
echo "All prestates successfully built and available in ${STATES_DIR}"
