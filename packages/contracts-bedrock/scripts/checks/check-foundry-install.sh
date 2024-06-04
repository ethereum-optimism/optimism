#!/usr/bin/env bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_BASE=$(dirname "$(dirname "$SCRIPT_DIR")")
MONOREPO_BASE=$(dirname "$(dirname "$CONTRACTS_BASE")")
VERSIONS_FILE="${MONOREPO_BASE}/versions.json"

if ! command -v forge &> /dev/null
then
  # shellcheck disable=SC2006
  echo "Is Foundry not installed? Consider installing via pnpm install:foundry" >&2
  exit 1
fi

# Check VERSIONS_FILE has expected foundry property
if ! jq -e '.foundry' "$VERSIONS_FILE" &> /dev/null; then
  echo "'foundry' is missing from $VERSIONS_FILE" >&2
  exit 1
fi

# Extract the expected foundry version from versions.json
EXPECTED_VERSION=$(jq -r '.foundry' "$VERSIONS_FILE" | cut -c 1-7)
if [ -z "$EXPECTED_VERSION" ]; then
  echo "Unable to extract Foundry version from $VERSIONS_FILE" >&2
  exit 1
fi

# Extract the installed forge version
INSTALLED_VERSION=$(forge --version | grep -o '[a-f0-9]\{7\}' | head -n 1)

# Compare the installed timestamp with the expected timestamp
if [ "$INSTALLED_VERSION" = "$EXPECTED_VERSION" ]; then
  echo "Foundry version matches the expected version."
else
  echo "Mismatch between installed Foundry version ($INSTALLED_VERSION) and expected version ($EXPECTED_VERSION)."
  echo "Your version of Foundry may either not be up to date, or it could be a later version."
  echo "Running pnpm update:foundry will install the expected version."
fi
