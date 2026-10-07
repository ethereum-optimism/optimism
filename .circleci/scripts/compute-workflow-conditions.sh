#!/usr/bin/env bash
# CircleCI adapter for the shared, provider-neutral workflow routing policy.
set -euo pipefail

case "${TRIGGER_SOURCE:?TRIGGER_SOURCE must be set}" in
  webhook) CI_EVENT=push ;;
  scheduled_pipeline) CI_EVENT=schedule ;;
  api) CI_EVENT=dispatch ;;
  *) echo "ERROR: unsupported CircleCI trigger '${TRIGGER_SOURCE}'" >&2; exit 1 ;;
esac
export CI_EVENT
export CI_BRANCH="${BRANCH:-}" CI_TAG="${TAG:-}" CI_SCHEDULE_NAME="${SCHEDULE_NAME:-}"
exec bash "$(dirname "${BASH_SOURCE[0]}")/../../ops/ci/runtime/compute-workflow-conditions.sh" "$@"
