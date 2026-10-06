#!/usr/bin/env bash
# Install only the pinned NUT generators and immutable Solidity compilers.
set -euo pipefail

retry_install() {
  local attempt status
  for attempt in 1 2 3 4 5; do
    if "$@"; then return 0; else status=$?; fi
    if [[ "${attempt}" == 5 ]]; then return "${status}"; fi
    sleep "$((2 ** attempt))"
  done
}

case "${1:-}" in
  solc)
    retry_install mise install svm-rs
    # Circle's four common versions plus 0.8.30 observed in both historical
    # generators. Forge still discovers the complete graph from original source.
    for version in 0.8.15 0.8.19 0.8.25 0.8.28 0.8.30; do
      if ! mise exec svm-rs -- svm which "${version}"; then
        retry_install mise exec svm-rs -- svm install "${version}"
      fi
    done
    ;;
  generators)
    [[ "${2:-}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ && "${3:-}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    retry_install mise install "forge@${2}" "just@${3}"
    ;;
  *) echo "Usage: $0 solc|generators FORGE_VERSION JUST_VERSION" >&2; exit 1 ;;
esac
