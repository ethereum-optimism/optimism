#!/usr/bin/env bash
# CircleCI compatibility entrypoint for shared parameter/path collection.
set -euo pipefail
export CI_BASE_REVISION="${BASE_REVISION:-develop}"
exec bash "$(dirname "${BASH_SOURCE[0]}")/../../ops/ci/collect-params.sh" "$@"
