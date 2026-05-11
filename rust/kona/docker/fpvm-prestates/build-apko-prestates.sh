#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 [kona-client|kona-client-int OUTPUT_DIR]" >&2
}

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
KONA_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
RUST_ROOT=$(cd "${KONA_ROOT}/.." && pwd)
REPO_ROOT=$(cd "${RUST_ROOT}/.." && pwd)

MELANGE_CONFIG="${SCRIPT_DIR}/kona-prestates.melange.yaml"
APKO_CONFIG="${SCRIPT_DIR}/kona-prestates.apko.yaml"

case "$#" in
  0)
    KONA_PRESTATE_VARIANTS="kona-client,kona-client-int"
    ;;
  2)
    case "$1" in
      kona-client | kona-client-int) ;;
      *)
        usage
        exit 2
        ;;
    esac
    KONA_PRESTATE_VARIANTS="$1"
    ;;
  *)
    usage
    exit 2
    ;;
esac

for bin in melange apko tar jq rsync; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "missing required command: ${bin}" >&2
    exit 1
  fi
done

TMP_DIR=$(mktemp -d)
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

export XDG_CACHE_HOME="${TMP_DIR}/cache"
mkdir -p "${XDG_CACHE_HOME}"

PACKAGES_DIR="${KONA_PRESTATE_PACKAGES_DIR:-${TMP_DIR}/packages}"
MELANGE_KEY="${MELANGE_KEY:-${TMP_DIR}/melange.rsa}"
if [ -z "${MELANGE_RUNNER:-}" ]; then
  if [ "$(uname -s)" = "Linux" ]; then
    MELANGE_RUNNER="bubblewrap"
  else
    MELANGE_RUNNER="docker"
  fi
fi

SOURCE_DIR="${TMP_DIR}/source"
mkdir -p "${SOURCE_DIR}"

stage_path() {
  local path=$1
  local source="${REPO_ROOT}/${path}"
  local dest="${SOURCE_DIR}/${path}"

  if [ ! -e "${source}" ]; then
    echo "required source path does not exist: ${path}" >&2
    exit 1
  fi

  mkdir -p "$(dirname "${dest}")"
  rsync -a \
    --exclude 'target/' \
    --exclude 'prestate-artifacts-cannon/' \
    --exclude 'prestate-artifacts-cannon-interop/' \
    "${source}" "${dest}"
}

stage_path go.mod
stage_path go.sum
stage_path mise.toml
stage_path justfiles/
stage_path cannon/
stage_path op-preimage/
stage_path op-service/
stage_path rust/
stage_path op-core/nuts/bundles/

KONA_CUSTOM_CONFIGS=false
if [ -n "${KONA_CUSTOM_CONFIGS_DIR:-}" ]; then
  if [ ! -d "${KONA_CUSTOM_CONFIGS_DIR}" ]; then
    echo "KONA_CUSTOM_CONFIGS_DIR=${KONA_CUSTOM_CONFIGS_DIR} is not a directory" >&2
    exit 1
  fi

  rsync -a "${KONA_CUSTOM_CONFIGS_DIR}/" "${SOURCE_DIR}/kona-custom-configs/"
  KONA_CUSTOM_CONFIGS=true
fi

ENV_FILE="${TMP_DIR}/melange.env"
{
  printf 'KONA_CUSTOM_CONFIGS=%s\n' "${KONA_CUSTOM_CONFIGS}"
  printf 'KONA_PRESTATE_VARIANTS=%s\n' "${KONA_PRESTATE_VARIANTS}"
} >"${ENV_FILE}"

if [ ! -f "${MELANGE_KEY}" ]; then
  melange keygen "${MELANGE_KEY}"
fi

rm -rf "${PACKAGES_DIR}"
mkdir -p "${PACKAGES_DIR}" "${TMP_DIR}/sbom"

melange build "${MELANGE_CONFIG}" \
  --arch x86_64 \
  --signing-key "${MELANGE_KEY}" \
  --source-dir "${SOURCE_DIR}" \
  --out-dir "${PACKAGES_DIR}" \
  --runner "${MELANGE_RUNNER}" \
  --env-file "${ENV_FILE}"

apko build "${APKO_CONFIG}" "kona-prestates:local" "${TMP_DIR}/kona-prestates.tar" \
  --arch x86_64 \
  --sbom-path "${TMP_DIR}/sbom" \
  -k "${MELANGE_KEY}.pub" \
  -r "${PACKAGES_DIR}"

APK_PATH=$(find "${PACKAGES_DIR}/x86_64" -maxdepth 1 -type f -name 'kona-prestates-*.apk' | sort | tail -1)
if [ -z "${APK_PATH}" ]; then
  echo "kona-prestates APK was not produced under ${PACKAGES_DIR}/x86_64" >&2
  exit 1
fi

EXTRACT_DIR="${TMP_DIR}/extract"
mkdir -p "${EXTRACT_DIR}"
tar -xzf "${APK_PATH}" -C "${EXTRACT_DIR}" usr/share/kona-prestates

copy_variant() {
  local variant=$1
  local output=$2
  local source="${EXTRACT_DIR}/usr/share/kona-prestates/${variant}"

  rm -rf "${output}"
  mkdir -p "${output}"
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
