#!/usr/bin/env bash

set -euo pipefail

# Generate the snapshots
pnpm snapshots

# Check if the generated snapshots are different from the committed snapshots
if git diff --exit-code snapshots > /dev/null; then
  [ -z "$(git ls-files --others --exclude-standard snapshots)" ] || exit 1
else
  exit 1
fi
