#!/usr/bin/env bash
# Component-specific tools and full-source prerequisites for contract shadows.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${REPO_ROOT}"

retry_download() {
  local attempt status
  for attempt in 1 2 3 4 5; do
    if "$@"; then return 0; else status=$?; fi
    if [[ "${attempt}" == 5 ]]; then return "${status}"; fi
    echo "Download failed; retrying attempt $((attempt + 1))/5." >&2
    sleep "$((2 ** attempt))"
  done
}

case "${1:-}" in
  tools)
    : "${RWX_ENV:?RWX_ENV must name the exported-environment directory}"
    # The shared bootstrap already trusted this disposable checkout. Keep each
    # version in mise.toml and install only the additional contract tools.
    retry_download mise install forge cast svm-rs
    PATH="$(mise bin-paths | paste -sd: -):${PATH}"
    export PATH
    # These are the exact compiler versions installed by the CircleCI command.
    for version in 0.8.15 0.8.19 0.8.25 0.8.28; do
      if ! svm which "${version}"; then
        retry_download svm install "${version}"
      fi
    done
    mkdir -p "${RWX_ENV}"
    printf '%s\n' "${PATH}" > "${RWX_ENV}/PATH"
    ;;
  source)
    mkdir -p .ci/contracts-prepare
    export GOCACHE="${REPO_ROOT}/.ci/go-cache/contracts/build"
    export GOMODCACHE="${REPO_ROOT}/.ci/go-cache/contracts/modules"
    mkdir -p "${GOCACHE}" "${GOMODCACHE}"
    # Keep the full source and Git state in this producer. In particular, do not
    # hide the gitlinks or nested Git directories behind a file-filter cache.
    git submodule sync --recursive
    retry_download git -c protocol.file.allow=never submodule update --init --recursive --jobs 8 \
      2>&1 | tee .ci/contracts-prepare/submodules.log
    git submodule status --recursive > .ci/contracts-prepare/submodule-status.txt
    retry_download go mod download 2>&1 | tee .ci/contracts-prepare/go-modules.log
    (cd packages/contracts-bedrock && just build-go-ffi) \
      2>&1 | tee .ci/contracts-prepare/go-ffi-build.log
    # The verdict runs this same checker against fresh Forge artifacts. Ship
    # the binary so it need not inherit a Go toolchain or the entire module cache.
    (cd packages/contracts-bedrock && go build -buildvcs=false \
      -o "${REPO_ROOT}/.ci/contracts-prepare/test-validation" ./scripts/checks/test-validation) \
      2>&1 | tee .ci/contracts-prepare/test-validation-build.log
    git rev-parse HEAD >.ci/contracts-prepare/commit-sha.txt
    # Deployer.gitCommit() explicitly supports this fallback when Git history
    # is absent (also used by packaged contract builds). Preserve the exact SHA.
    cp .ci/contracts-prepare/commit-sha.txt packages/contracts-bedrock/.gitcommit
    bash ops/ci/rwx-source-archive.sh .ci/contracts-prepare/source.tar.gz
    ;;
  *)
    echo "Usage: $0 tools|source" >&2
    exit 1
    ;;
esac
