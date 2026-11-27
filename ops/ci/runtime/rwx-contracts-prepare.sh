#!/usr/bin/env bash
# Install the pinned Foundry tools and solc versions for the contract tasks.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
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

if [[ "${1:-}" != tools ]]; then
  echo "Usage: $0 tools" >&2
  exit 1
fi
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
