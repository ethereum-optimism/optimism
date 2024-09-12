#!/bin/bash
set -euo pipefail

SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
COMPAT_DIR="${SCRIPTS_DIR}/../temp/compat"

TESTNAME="${1?Must specify compat file to run}"
BASEURL="${2:-https://github.com/ethereum-optimism/chain-test-data/releases/download/2024-09-01}"

URL="${BASEURL}/${TESTNAME}.tar.bz"

mkdir -p "${COMPAT_DIR}"
curl --etag-save "${COMPAT_DIR}/${TESTNAME}-etag.txt" --etag-compare "${COMPAT_DIR}/${TESTNAME}-etag.txt" -L --fail -o "${COMPAT_DIR}/${TESTNAME}.tar.bz" "${URL}"
tar jxf "${COMPAT_DIR}/${TESTNAME}.tar.bz" -C "${COMPAT_DIR}"
# shellcheck disable=SC2046
"${SCRIPTS_DIR}/../bin/op-program" --data.format=pebble $(cat "${COMPAT_DIR}/${TESTNAME}/args.txt")
