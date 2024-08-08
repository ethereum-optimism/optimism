#!/usr/bin/env bash

set -euo pipefail

# Generate the snapshots
just snapshots

# Check if the generated `snapshots` or `test/kontrol` files are different from the committed versions
if git diff --exit-code snapshots test/kontrol > /dev/null; then
  [ -z "$(git ls-files --others --exclude-standard snapshots test/kontrol)" ] || exit 1
else
  exit 1
fi
