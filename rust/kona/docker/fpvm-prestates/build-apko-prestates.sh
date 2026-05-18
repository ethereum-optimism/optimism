#!/usr/bin/env bash
# Build reproducible kona-client cannon prestates as composed apko/melange
# packages:
#
#   rust-nightly-2026-02-20  (pinned nightly toolchain APK, hash-pinned fetch)
#   cannon                   (Go MIPS64 emulator; uses: go/build)
#   kona-client-elf          (Rust MIPS64 ELFs; uses: cargo/build-no-auditable,
#                             local pipeline override; depends rust-nightly)
#   kona-prestates           (cannon-runs-over-ELFs glue; depends cannon +
#                             kona-client-elf)
#
# apko then composes the final image from kona-prestates alone. The script
# extracts the prestate artifacts back into the existing checked-in output
# directories so downstream tooling does not need to change.
set -euo pipefail

usage() {
  echo "usage: $0 [kona-client|kona-client-int OUTPUT_DIR]" >&2
}

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
KONA_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
RUST_ROOT=$(cd "${KONA_ROOT}/.." && pwd)
REPO_ROOT=$(cd "${RUST_ROOT}/.." && pwd)

CANNON_CONFIG="${SCRIPT_DIR}/cannon.melange.yaml"
NIGHTLY_CONFIG="${SCRIPT_DIR}/rust-nightly-2026-02-20.melange.yaml"
ELF_CONFIG="${SCRIPT_DIR}/kona-client-elf.melange.yaml"
PRESTATES_MELANGE="${SCRIPT_DIR}/kona-prestates.melange.yaml"
APKO_CONFIG="${SCRIPT_DIR}/kona-prestates.apko.yaml"
PIPELINES_DIR="${SCRIPT_DIR}/melange-pipelines"

case "$#" in
  0)
    KONA_PRESTATE_VARIANTS="kona-client,kona-client-int"
    ;;
  2)
    case "$1" in
      kona-client | kona-client-int) ;;
      *) usage; exit 2 ;;
    esac
    KONA_PRESTATE_VARIANTS="$1"
    ;;
  *) usage; exit 2 ;;
esac

for bin in melange apko tar jq; do
  command -v "${bin}" >/dev/null 2>&1 || { echo "missing required command: ${bin}" >&2; exit 1; }
done

TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

PACKAGES_DIR="${KONA_PRESTATE_PACKAGES_DIR:-${TMP_DIR}/packages}"
MELANGE_KEY="${MELANGE_KEY:-${TMP_DIR}/melange.rsa}"
CACHE_DIR="${KONA_PRESTATE_CACHE_DIR:-${TMP_DIR}/cache}"
if [ -z "${MELANGE_RUNNER:-}" ]; then
  if [ "$(uname -s)" = "Linux" ]; then MELANGE_RUNNER="bubblewrap"; else MELANGE_RUNNER="docker"; fi
fi

mkdir -p "${PACKAGES_DIR}" "${CACHE_DIR}"
if [ ! -f "${MELANGE_KEY}" ]; then
  melange keygen "${MELANGE_KEY}"
fi

# stage_paths SOURCE_DIR PATH...
# rsync each PATH (relative to REPO_ROOT) into SOURCE_DIR, excluding build
# output directories.
stage_paths() {
  local src_dir=$1; shift
  for path in "$@"; do
    local source="${REPO_ROOT}/${path}"
    local dest_parent="${src_dir}/$(dirname "${path}")"
    if [ ! -e "${source}" ]; then echo "missing source path: ${path}" >&2; exit 1; fi
    mkdir -p "${dest_parent}"
    rsync -a \
      --exclude 'target/' \
      --exclude 'prestate-artifacts-cannon/' \
      --exclude 'prestate-artifacts-cannon-interop/' \
      "${source}" "${dest_parent}/"
  done
}

run_melange() {
  local config=$1; shift
  melange build "${config}" \
    --arch x86_64 \
    --runner "${MELANGE_RUNNER}" \
    --signing-key "${MELANGE_KEY}" \
    --out-dir "${PACKAGES_DIR}" \
    --cache-dir "${CACHE_DIR}" \
    --keyring-append "${MELANGE_KEY}.pub" \
    --repository-append "${PACKAGES_DIR}" \
    --pipeline-dir "${PIPELINES_DIR}" \
    "$@"
}

# 1. Cannon (Go).
CANNON_SRC="${TMP_DIR}/cannon-src"
mkdir -p "${CANNON_SRC}"
stage_paths "${CANNON_SRC}" go.mod go.sum cannon op-service op-preimage
run_melange "${CANNON_CONFIG}" --source-dir "${CANNON_SRC}"

# 2. Rust nightly toolchain. No source required; the package fetches signed
#    tarballs from static.rust-lang.org by hash.
run_melange "${NIGHTLY_CONFIG}"

# 3. kona-client-elf (Rust MIPS64).
ELF_SRC="${TMP_DIR}/kona-src"
mkdir -p "${ELF_SRC}"
stage_paths "${ELF_SRC}" rust op-core/nuts/bundles

if [ -n "${KONA_CUSTOM_CONFIGS_DIR:-}" ]; then
  if [ ! -d "${KONA_CUSTOM_CONFIGS_DIR}" ]; then
    echo "KONA_CUSTOM_CONFIGS_DIR=${KONA_CUSTOM_CONFIGS_DIR} is not a directory" >&2
    exit 1
  fi
  rsync -a "${KONA_CUSTOM_CONFIGS_DIR}/" "${ELF_SRC}/kona-custom-configs/"
fi
run_melange "${ELF_CONFIG}" --source-dir "${ELF_SRC}"

# 4. Compose prestate artifacts. No source dir; runs cannon over installed ELFs.
ENV_FILE="${TMP_DIR}/melange.env"
printf 'KONA_PRESTATE_VARIANTS=%s\n' "${KONA_PRESTATE_VARIANTS}" >"${ENV_FILE}"
run_melange "${PRESTATES_MELANGE}" --env-file "${ENV_FILE}"

# 5. Build the apko image bundling kona-prestates only.
apko build "${APKO_CONFIG}" "kona-prestates:local" "${TMP_DIR}/kona-prestates.tar" \
  --arch x86_64 \
  -k "${MELANGE_KEY}.pub" \
  -r "${PACKAGES_DIR}"

# 6. Extract prestate artifacts from the kona-prestates apk into the existing
#    checked-in output layout.
APK_PATH=$(find "${PACKAGES_DIR}/x86_64" -maxdepth 1 -type f -name 'kona-prestates-*.apk' | sort | tail -1)
if [ -z "${APK_PATH}" ]; then
  echo "kona-prestates APK was not produced under ${PACKAGES_DIR}/x86_64" >&2
  exit 1
fi
EXTRACT_DIR="${TMP_DIR}/extract"
mkdir -p "${EXTRACT_DIR}"
tar -xzf "${APK_PATH}" -C "${EXTRACT_DIR}" usr/share/kona-prestates

copy_variant() {
  local variant=$1 output=$2
  local source="${EXTRACT_DIR}/usr/share/kona-prestates/${variant}"
  rm -rf "${output}"; mkdir -p "${output}"
  cp "${source}/prestate.bin.gz" "${output}/prestate.bin.gz"
  cp "${source}/prestate-proof.json" "${output}/prestate-proof.json"
  cp "${source}/meta.json" "${output}/meta.json"
  cp "${source}/kona-client-elf" "${output}/kona-client-elf"
  local hash
  hash=$(jq -r .pre "${output}/prestate-proof.json")
  cp "${output}/prestate.bin.gz" "${output}/${hash}.bin.gz"
  echo "Prestate for ${variant}: ${hash}"
}

if [ "$#" -eq 2 ]; then
  copy_variant "$1" "$2"
else
  copy_variant kona-client "${KONA_ROOT}/prestate-artifacts-cannon"
  copy_variant kona-client-int "${KONA_ROOT}/prestate-artifacts-cannon-interop"
fi
