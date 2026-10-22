#!/usr/bin/env bash
# Prepare the pinned tools needed by the RWX pilot on its Ubuntu base.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"
: "${RWX_ENV:?RWX_ENV must name the exported-environment directory}"

# Language toolchains are independent layers. In particular, Go/Foundry tasks
# must not download both Rust toolchains or the Go linter to run their tests.
case "${1:-common}" in
  common)
    bash ops/ci/runtime/apt-install.sh \
      build-essential ca-certificates pkg-config unzip zip
    # Trust only the disposable RWX checkout; local development is unchanged.
    mise trust --yes mise.toml
    mise install just jq yq python
    ;;
  go) mise install go gotestsum ;;
  # depguard tests execute go list at runtime; keep the pinned Go toolchain.
  go-runtime) mise install go gotestsum ;;
  go-lint) mise install go golangci-lint ;;
  rust) mise install rust ;;
  *) echo "Usage: $0 [common|go|go-runtime|go-lint|rust]" >&2; exit 1 ;;
esac
mkdir -p "${RWX_ENV}"
printf '%s:%s\n' "$(mise bin-paths | paste -sd: -)" "${PATH}" >"${RWX_ENV}/PATH"
# Later mise exec calls must use this prepared toolset rather than installing
# unrelated tools from the root configuration in every verdict task.
printf 'false\n' >"${RWX_ENV}/MISE_AUTO_INSTALL"
printf 'false\n' >"${RWX_ENV}/MISE_EXEC_AUTO_INSTALL"
