#!/usr/bin/env bash
# Prepare the pinned tools needed by the RWX pilot on its Ubuntu base.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${REPO_ROOT}"
: "${RWX_ENV:?RWX_ENV must name the exported-environment directory}"

# Reuse the existing retrying installer until system preparation is migrated.
bash .circleci/scripts/apt-install.sh \
  build-essential ca-certificates pkg-config unzip zip

# This is a distinct RWX task snapshot, not CircleCI's shared write-once mise
# cache. Install only pilot tools while keeping all versions in mise.toml.
# Bootstrap trust only inside the disposable RWX task (RWX_ENV is required
# above). Local development continues to require the user-run mise trust step.
mise trust --yes mise.toml
mise install go golangci-lint just jq yq python rust
mkdir -p "${RWX_ENV}"
printf '%s:%s\n' "$(mise bin-paths | paste -sd: -)" "${PATH}" >"${RWX_ENV}/PATH"
# Later mise exec calls must use this prepared toolset rather than installing
# unrelated tools from the root configuration in every verdict task.
printf 'false\n' >"${RWX_ENV}/MISE_AUTO_INSTALL"
printf 'false\n' >"${RWX_ENV}/MISE_EXEC_AUTO_INSTALL"
