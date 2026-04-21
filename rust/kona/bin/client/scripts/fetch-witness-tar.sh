#!/usr/bin/env bash
# Download a preimage witness tar from chain-test-data releases with etag caching.
# Mirrors op-program/scripts/run-compat.sh — skips redownload when the release asset
# is unchanged so local re-runs are fast.
#
# Usage: fetch-witness-tar.sh <tar_name> <base_url>
#
# Invoked via the `prepare-witness-data` just recipe, which supplies the
# canonical tar name and release URL. The tar is written to
# rust/kona/bin/client/testdata/<tar_name>, alongside an etag sidecar that
# enables conditional GETs on subsequent invocations.
set -o errexit -o nounset -o pipefail

SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
TESTDATA_DIR="${SCRIPTS_DIR}/../testdata"

TAR_NAME="${1?Must specify tar name (e.g. op-sepolia-42427365-witness.tar.zst)}"
BASE_URL="${2?Must specify chain-test-data release base URL (e.g. https://github.com/ethereum-optimism/chain-test-data/releases/download/kona-YYYY-MM-DD)}"

URL="${BASE_URL}/${TAR_NAME}"
TAR_PATH="${TESTDATA_DIR}/${TAR_NAME}"
ETAG_PATH="${TESTDATA_DIR}/${TAR_NAME}.etag"

mkdir -p "${TESTDATA_DIR}"
curl --etag-save "${ETAG_PATH}" --etag-compare "${ETAG_PATH}" \
  -L --fail -o "${TAR_PATH}" "${URL}"
