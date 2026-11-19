#!/usr/bin/env bash
set -euo pipefail

if [[ "${IMAGE_ARCH:-}" != "amd64" ]]; then
  echo "::notice::genesis smoke test runs on amd64 only, skipping ${IMAGE_ARCH:-unknown}"
  exit 0
fi

: "${IMAGE:?IMAGE must be set}"
exec "$(dirname "${BASH_SOURCE[0]}")/genesis-smoke-test.sh" "$IMAGE"
