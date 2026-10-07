#!/usr/bin/env bash
# Preserve the setup-pipeline command while exercising both provider entrypoints.
set -euo pipefail
exec bash "$(dirname "${BASH_SOURCE[0]}")/../../ops/ci/migration/tests/test-circle-adapter.sh" "$@"
