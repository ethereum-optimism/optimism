#!/usr/bin/env bash
# Fail unless the program in a final Cannon state exited with code 0.
#
# `cannon run` exits 0 whatever the program's exit code is. The client rejects
# an invalid claim by exiting non-zero, so that result is visible only in the
# final state.
#
# Usage: require-cannon-success.sh <final_state.bin.gz>
#
# Requires `cannon` and `jq` on PATH.
set -o errexit -o nounset -o pipefail

FINAL_STATE="${1?Must specify the final Cannon state written by cannon run --output}"

STATUS=$(cannon witness --input "${FINAL_STATE}" | jq -r '"\(.exited) \(.exitCode)"')
if [[ "${STATUS}" != "true 0" ]]; then
  echo "error: the client did not exit with code 0 in Cannon (exited, exit code: ${STATUS})" >&2
  exit 1
fi
echo "The client exited with code 0 in Cannon."
