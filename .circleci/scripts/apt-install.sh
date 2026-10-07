#!/usr/bin/env bash
# Circle adapter for the shared package installer.
set -euo pipefail
exec bash "$(dirname "${BASH_SOURCE[0]}")/../../ops/ci/runtime/apt-install.sh" "$@"
