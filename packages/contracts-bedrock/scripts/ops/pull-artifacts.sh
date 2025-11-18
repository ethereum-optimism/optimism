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

download_and_extract() {
  local archive_name=$1

  echoerr "> Downloading..."
  curl --fail --location --connect-timeout 30 --max-time 300 --tlsv1.2 -o "$archive_name" "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name"
  echoerr "> Done."

  echoerr "> Cleaning up existing artifacts..."
  rm -rf artifacts
  rm -rf forge-artifacts
  rm -rf cache
  echoerr "> Done."

  echoerr "> Extracting artifacts..."
  if [[ "$archive_name" == *.tar.zst ]]; then
    zstd -dc "$archive_name" | tar -xf - --exclude='*..*'
  else
    tar -xzvf "$archive_name" --exclude='*..*'
  fi
  echoerr "> Done."

  echoerr "> Cleaning up."
  rm "$archive_name"
  echoerr "> Done."
  exit 0
}

# Check for help flag
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  usage
fi

# Check for fallback-to-latest flag
USE_LATEST_FALLBACK=false
if [ "${1:-}" = "--fallback-to-latest" ]; then
  USE_LATEST_FALLBACK=true
  echoerr "> Fallback to latest enabled"
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
    download_and_extract "$archive_name_zst"
  fi

  # Try latest fallback if enabled
  if [ "$USE_LATEST_FALLBACK" = true ]; then
    echoerr "> Exact checksum not found, trying latest artifacts..."
    archive_name_zst="artifacts-v1-latest.tar.zst"
    exists_latest_zst=$(curl -s -o /dev/null --fail -LI "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name_zst" || echo "fail")

    if [ "$exists_latest_zst" != "fail" ]; then
      download_and_extract "$archive_name_zst"
    fi
  fi
fi

archive_name_gz="artifacts-v1-$checksum.tar.gz"
exists_gz=$(curl -s -o /dev/null --fail -LI "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name_gz" || echo "fail")

if [ "$exists_gz" == "fail" ]; then
  # Try latest fallback if enabled
  if [ "$USE_LATEST_FALLBACK" = true ]; then
    echoerr "> Exact checksum not found, trying latest artifacts..."
    archive_name_gz="artifacts-v1-latest.tar.gz"
    exists_latest_gz=$(curl -s -o /dev/null --fail -LI "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name_gz" || echo "fail")

    if [ "$exists_latest_gz" == "fail" ]; then
      echoerr "> No existing artifacts found (including latest), exiting."
      exit 0
    fi

    echoerr "> Found latest .tar.gz artifacts."
  else
    echoerr "> No existing artifacts found, exiting."
    exit 0
  fi
fi

download_and_extract "$archive_name_gz"