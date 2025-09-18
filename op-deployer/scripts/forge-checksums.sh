#!/usr/bin/env bash
set -euo pipefail

# Usage: ./forge-checksums.sh v1.3.1
if [[ $# -ne 1 ]]; then
  echo "usage: $0 vX.Y.Z" >&2
  exit 1
fi
VER="$1"

# Matrix of supported OS/arch combos
pairs=(
  "darwin_amd64"
  "darwin_arm64"
  "linux_amd64"
  "linux_arm64"
  "alpine_amd64"
  "alpine_arm64"
)

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JSON_PATH="${SCRIPT_DIR}/../pkg/deployer/forge/version.json"

# Temp workspace
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

declare -A sums

# Download each tarball and compute sha256 over the tar.gz (matches code path)
for pair in "${pairs[@]}"; do
  os="${pair%_*}"
  arch="${pair#*_}"
  url="https://github.com/foundry-rs/foundry/releases/download/${VER}/foundry_${VER}_${os}_${arch}.tar.gz"
  out="${TMP_DIR}/foundry_${VER}_${pair}.tar.gz"

  echo "--------------------------------"
  echo "Processing ${pair}..."

  echo "Downloading ${url}"
  curl -fsSL --retry 3 --retry-delay 1 -o "${out}" "${url}"

  echo "Computing checksum"
  sha="$(shasum -a 256 "${out}" | awk '{print $1}')"
  echo "Checksum for ${pair}: ${sha}"
  sums["${pair}"]="${sha}"
done

echo "--------------------------------"
echo "Done computing checksums"
echo "Writing results to ${JSON_PATH}"

# Write version.json
mkdir -p "$(dirname "${JSON_PATH}")"
{
  printf '{\n'
  printf '  "forge": "%s",\n' "${VER}"
  printf '  "checksums": {\n'
  for ((i=0; i<${#pairs[@]}; i++)); do
    key="${pairs[$i]}"
    val="${sums[$key]}"
    if (( i < ${#pairs[@]} - 1 )); then
      printf '    "%s": "%s",\n' "${key}" "${val}"
    else
      printf '    "%s": "%s"\n' "${key}" "${val}"
    fi
  done
  printf '  }\n'
  printf '}\n'
} > "${JSON_PATH}"

echo "Success!"
