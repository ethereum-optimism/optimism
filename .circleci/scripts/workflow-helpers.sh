#!/usr/bin/env bash
# Compatibility entrypoint for callers sourcing the former CircleCI helper.
# shellcheck disable=SC1091  # shared helper resolved relative to this wrapper
source "$(dirname "${BASH_SOURCE[0]}")/../../ops/ci/runtime/workflow-helpers.sh"
