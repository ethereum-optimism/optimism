#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "Usage: $0 [--force|-f]"
  echo ""
  echo "Publish contract artifacts to GCS with optional zstd compression."
  echo ""
  echo "Options:"
  echo "  --force, -f    Force upload even if artifacts already exist"
  echo "  --help, -h     Show this help message"
  echo ""
  echo "If zstd is available, creates both .tar.gz and .tar.zst files."
  echo "Otherwise, creates only .tar.gz with a warning about future zstd requirement."
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
DEPLOY_BUCKET="oplabs-contract-artifacts"

cd "$CONTRACTS_DIR"

# Check for force flag
FORCE=false
if [ "${1:-}" = "--force" ] || [ "${1:-}" = "-f" ]; then
  FORCE=true
  echoerr "> Force mode enabled - will overwrite existing artifacts"
fi

if command -v zstd >/dev/null 2>&1; then
  HAS_ZSTD=true
  echoerr "> zstd found, will create both .tar.gz and .tar.zst files"
else
  HAS_ZSTD=false
  echoerr "> zstd not found, will create only .tar.gz files"
  echoerr "> WARNING: zstd not available. In the future, only zstd will be supported."
fi

# ensure that artifacts exists and is non-empty
if [ ! -d "forge-artifacts" ] || [ -z "$(ls -A forge-artifacts)" ]; then
  echoerr "> No forge-artifacts directory found."
  exit 1
fi

if [ ! -d "artifacts" ] || [ -z "$(ls -A artifacts)" ]; then
  echoerr "> No artifacts directory found."
  exit 1
fi

checksum=$(bash scripts/ops/calculate-checksum.sh)

echoerr "> Checksum: $checksum"
echoerr "> Checking for existing artifacts..."

if [ "$HAS_ZSTD" = true ]; then
  upload_url_zst="https://storage.googleapis.com/$DEPLOY_BUCKET/artifacts-v1-$checksum.tar.zst"
  exists_zst=$(curl -s -o /dev/null --fail -LI "$upload_url_zst" || echo "fail")
  if [ "$exists_zst" != "fail" ] && [ "$FORCE" = false ]; then
    echoerr "> Existing artifacts found (.tar.zst), nothing to do. Use --force to overwrite."
    exit 0
  fi
fi

upload_url_gz="https://storage.googleapis.com/$DEPLOY_BUCKET/artifacts-v1-$checksum.tar.gz"
exists_gz=$(curl -s -o /dev/null --fail -LI "$upload_url_gz" || echo "fail")

if [ "$exists_gz" != "fail" ] && [ "$FORCE" = false ]; then
  echoerr "> Existing artifacts found (.tar.gz), nothing to do. Use --force to overwrite."
  exit 0
fi

echoerr "> Archiving artifacts..."

# use gtar on darwin
if [[ "$OSTYPE" == "darwin*" ]]; then
  tar="gtar"
else
  tar="tar"
fi

rm -f COMMIT
commit=$(git rev-parse HEAD)
echo "$commit" > COMMIT


if [ "$HAS_ZSTD" = true ]; then
  echoerr "> Compressing artifacts (.tar.gz and .tar.zst)..."

  # Create intermediate tar file first for reliable compression
  temp_tar="artifacts-v1-$checksum.tar"
  tar_args="artifacts forge-artifacts COMMIT"
  if [ -d "cache" ]; then
    tar_args="$tar_args cache"
    echoerr "> Including cache directory in archive"
  else
    echoerr "> Cache directory not found, excluding from archive"
  fi
  "$tar" -cf "$temp_tar" "$tar_args"

  archive_name_gz="artifacts-v1-$checksum.tar.gz"
  archive_name_zst="artifacts-v1-$checksum.tar.zst"

  gzip -9 < "$temp_tar" > "$archive_name_gz" &
  gz_pid=$!

  zstd --ultra -22 -f "$temp_tar" -o "$archive_name_zst" &
  zst_pid=$!

  wait "$gz_pid"
  wait "$zst_pid"

  rm "$temp_tar"

  du -sh "$archive_name_gz" | awk '{$1=$1};1' # trim leading whitespace
  echoerr "> Created .tar.gz archive"
  du -sh "$archive_name_zst" | awk '{$1=$1};1' # trim leading whitespace
  echoerr "> Created .tar.zst archive"

  # Compare file sizes in MB
  gz_size=$(stat -f%z "$archive_name_gz" 2>/dev/null || stat -c%s "$archive_name_gz" 2>/dev/null || echo "0")
  zst_size=$(stat -f%z "$archive_name_zst" 2>/dev/null || stat -c%s "$archive_name_zst" 2>/dev/null || echo "0")

  if [ "$gz_size" -gt 0 ] && [ "$zst_size" -gt 0 ]; then
    gz_mb=$(awk "BEGIN {printf \"%.2f\", $gz_size / 1048576}")
    zst_mb=$(awk "BEGIN {printf \"%.2f\", $zst_size / 1048576}")
    savings=$((gz_size - zst_size))
    savings_percent=$((100 * savings / gz_size))
    echoerr "> Size comparison: .tar.gz=${gz_mb}MB, .tar.zst=${zst_mb}MB (${savings_percent}% smaller)"
  fi
else
  echoerr "> Compressing artifacts (.tar.gz)..."
  archive_name_gz="artifacts-v1-$checksum.tar.gz"
  tar_args="artifacts forge-artifacts COMMIT"
  if [ -d "cache" ]; then
    tar_args="$tar_args cache"
    echoerr "> Including cache directory in archive"
  else
    echoerr "> Cache directory not found, excluding from archive"
  fi
  "$tar" -czf "$archive_name_gz" "$tar_args"
  du -sh "$archive_name_gz" | awk '{$1=$1};1' # trim leading whitespace
  echoerr "> Created .tar.gz archive"
fi

echoerr "> Done."

echoerr "> Uploading artifacts to GCS..."

# Force single-stream upload to improve reliability
gcloud config set storage/parallel_composite_upload_enabled False
if [ "$HAS_ZSTD" = true ]; then
  gcloud --verbosity="info" storage cp "$archive_name_gz" "$archive_name_zst" "gs://$DEPLOY_BUCKET/"
  echoerr "> Uploaded to: $upload_url_gz"
  echoerr "> Uploaded to: $upload_url_zst"
else
  gcloud --verbosity="info" storage cp "$archive_name_gz" "gs://$DEPLOY_BUCKET/$archive_name_gz"
  echoerr "> Uploaded to: $upload_url_gz"
fi

echoerr "> Done."

rm "$archive_name_gz"
if [ "$HAS_ZSTD" = true ]; then
  rm "$archive_name_zst"
fi
rm COMMIT
