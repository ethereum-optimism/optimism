#!/usr/bin/env bash

set -euo pipefail

echoerr() {
  echo "$@" 1>&2
}

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_DIR="$SCRIPT_DIR/../.."

cd "$CONTRACTS_DIR"

checksum=$(bash scripts/ops/calculate-checksum.sh)
archive_name="artifacts-v1-$checksum.tar.gz"

echoerr "> Checking for existing artifacts..."
exists=$(curl -s -o /dev/null --fail -LI "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name" || echo "fail")

if [ "$exists" == "fail" ]; then
  echoerr "> No existing artifacts found, exiting."
  exit 0
fi

echoerr "> Cleaning up existing artifacts..."
rm -rf artifacts
rm -rf forge-artifacts
rm -rf cache
echoerr "> Done."

echoerr "> Found existing artifacts. Downloading..."
curl -o "$archive_name" "https://storage.googleapis.com/oplabs-contract-artifacts/$archive_name"
echoerr "> Done."

echoerr "> Extracting existing artifacts..."
tar -xzvf "$archive_name"
echoerr "> Done."

echoerr "> Cleaning up."
rm "$archive_name"
echoerr "> Done."