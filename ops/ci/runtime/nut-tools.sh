#!/usr/bin/env bash
# Install the tools that the historical NUT bundle generators need beyond the
# contract tool layer. nut-provenance.py calls this script.
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
    # The contract tool layer installs the four versions that CircleCI installs.
    # The historical generators were also observed to compile with 0.8.30.
    # Install it here, so that the verdict task downloads no compiler.
    if ! svm which 0.8.30; then
      retry_install svm install 0.8.30
    fi
    ;;
  generators)
    if ! [[ "${2:-}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ && "${3:-}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "The recorded commit must pin exact Forge and Just versions: '${2:-}' '${3:-}'" >&2
      exit 1
    fi
    retry_install mise install "forge@${2}" "just@${3}"
    ;;
  *)
    echo "Usage: $0 solc|generators FORGE_VERSION JUST_VERSION" >&2
    exit 1
    ;;
esac
