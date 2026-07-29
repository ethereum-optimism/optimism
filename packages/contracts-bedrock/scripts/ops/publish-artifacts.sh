#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "Usage: $0 [--force|-f]"
  echo ""
  echo "Publish contract artifacts to GCS using op-deployer's build process."
  echo ""
  echo "Options:"
  echo "  --force, -f    Force upload even if artifacts already exist"
  echo "  --help, -h     Show this help message"
  echo ""
  echo "Uses 'just build-contracts' and 'just copy-contract-artifacts' from op-deployer."
  exit 0
}

echoerr() {
  echo "$@" 1>&2
}

if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  usage
fi

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
CONTRACTS_DIR="$SCRIPT_DIR/../.."
DEPLOY_BUCKET="oplabs-contract-artifacts"
BUCKET_URL="https://storage.googleapis.com/$DEPLOY_BUCKET"

# Resolve paths
ROOT_DIR=$(cd -- "$CONTRACTS_DIR/../.." &> /dev/null && pwd)
OP_DEPLOYER_DIR="$ROOT_DIR/op-deployer"

if [ ! -d "$OP_DEPLOYER_DIR" ]; then
  echoerr "> ERROR: op-deployer not found at $OP_DEPLOYER_DIR"
  exit 1
fi

FORCE=false
if [ "${1:-}" = "--force" ] || [ "${1:-}" = "-f" ]; then
  FORCE=true
  echoerr "> Force mode enabled - will overwrite existing artifacts"
fi

checksum=$(bash "$CONTRACTS_DIR/scripts/ops/calculate-checksum.sh")

archive_name_zst="artifacts-v1-$checksum.tar.zst"
upload_url_zst="$BUCKET_URL/$archive_name_zst"

echoerr "> Checksum: $checksum"
echoerr "> Checking for existing artifacts..."

exists_zst=$(curl -s -o /dev/null --fail -LI "$upload_url_zst" || echo "fail")
if [ "$exists_zst" != "fail" ] && [ "$FORCE" = false ]; then
  echoerr "> Existing artifacts found (.tar.zst), nothing to do. Use --force to overwrite."
  exit 0
fi

echoerr "> Building contracts and creating artifacts..."

cd "$OP_DEPLOYER_DIR"

echoerr "> Running 'just build-contracts'..."
just build-contracts

echoerr "> Running 'just copy-contract-artifacts'..."
just copy-contract-artifacts

ARTIFACTS_TZST="./pkg/deployer/artifacts/forge-artifacts/artifacts.tzst"
if [ ! -f "$ARTIFACTS_TZST" ]; then
  echoerr "> ERROR: Failed to create artifacts.tzst"
  exit 1
fi

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

cp "$ARTIFACTS_TZST" "$TEMP_DIR/$archive_name_zst"
du -sh "$TEMP_DIR/$archive_name_zst" | awk '{$1=$1};1'
echoerr "> Created .tar.zst archive"

echoerr "> Uploading artifacts to $BUCKET_URL..."

# Force single-stream upload to improve reliability
gcloud config set storage/parallel_composite_upload_enabled False

gcloud --verbosity="info" storage cp "$TEMP_DIR/$archive_name_zst" "gs://$DEPLOY_BUCKET/$archive_name_zst"
echoerr "> Uploaded to: $upload_url_zst"

echoerr "> Uploading as 'latest' for PR fallback..."
gcloud --verbosity="info" storage cp "gs://$DEPLOY_BUCKET/$archive_name_zst" "gs://$DEPLOY_BUCKET/artifacts-v1-latest.tar.zst"
echoerr "> Uploaded to: https://storage.googleapis.com/$DEPLOY_BUCKET/artifacts-v1-latest.tar.zst"

echoerr "> Done."
