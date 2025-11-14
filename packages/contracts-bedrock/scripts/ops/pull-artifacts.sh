#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "Usage: $0"
  echo ""
  echo "Download contract artifacts from GCS, preferring zstd if available."
  echo ""
  echo "If zstd is available, downloads .tar.zst files when present."
  echo "Otherwise, falls back to .tar.gz files."
  exit 0
}

echoerr() {
  echo "$@" 1>&2
}

# Check for help flag
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  usage
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_DIR="$SCRIPT_DIR/../.."

cd "$CONTRACTS_DIR"

if command -v zstd >/dev/null 2>&1; then
  HAS_ZSTD=true
  echoerr "> zstd found, will prefer .tar.zst files"
else
  HAS_ZSTD=false
  echoerr "> zstd not found, will prefer .tar.gz files"
fi

checksum=$(bash scripts/ops/calculate-checksum.sh)

echoerr "> Checking for existing artifacts..."

if [ "$HAS_ZSTD" = true ]; then
  archive_name_zst="artifacts-v1-$checksum.tar.zst"
  exists_zst=$(curl -s -o /dev/null --fail -LI "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name_zst" || echo "fail")
  
  if [ "$exists_zst" != "fail" ]; then
    echoerr "> Found .tar.zst artifacts. Downloading..."
    curl -o "$archive_name_zst" "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name_zst"
    echoerr "> Done."
    
    echoerr "> Cleaning up existing artifacts..."
    rm -rf artifacts
    rm -rf forge-artifacts
    rm -rf cache
    echoerr "> Done."
    
    echoerr "> Extracting existing artifacts..."
    zstd -dc "$archive_name_zst" | tar -xf -
    echoerr "> Done."
    
    echoerr "> Cleaning up."
    rm "$archive_name_zst"
    echoerr "> Done."
    exit 0
  fi
fi

archive_name_gz="artifacts-v1-$checksum.tar.gz"
exists_gz=$(curl -s -o /dev/null --fail -LI "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name_gz" || echo "fail")

if [ "$exists_gz" == "fail" ]; then
  echoerr "> No existing artifacts found, exiting."
  exit 0
fi

if [ "$HAS_ZSTD" = true ]; then
  echoerr "> Only .tar.gz artifacts available (zstd format not found)."
else
  echoerr "> Found .tar.gz artifacts (zstd not available)."
fi

echoerr "> Cleaning up existing artifacts..."
rm -rf artifacts
rm -rf forge-artifacts
rm -rf cache
echoerr "> Done."

echoerr "> Downloading artifacts..."
curl -o "$archive_name_gz" "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name_gz"
echoerr "> Done."

echoerr "> Extracting existing artifacts..."
tar -xzvf "$archive_name_gz"
echoerr "> Done."

echoerr "> Cleaning up."
rm "$archive_name_gz"
echoerr "> Done."