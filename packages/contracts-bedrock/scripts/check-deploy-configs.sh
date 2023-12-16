#!/usr/bin/env bash

# This script is used to check for valid deploy configs.
# It should check all configs and return a non-zero exit code if any of them are invalid.
# getting-started.json isn't valid JSON so its skipped.

code=0

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_BASE=$(dirname "$SCRIPT_DIR")
MONOREPO_BASE=$(dirname "$(dirname "$CONTRACTS_BASE")")

for config in "$CONTRACTS_BASE"/deploy-config/*.json; do
    # shellcheck disable=SC2086
    if ! go run "$MONOREPO_BASE/op-chain-ops/cmd/check-deploy-config/main.go" --path "$config"; then
        code=1
    fi
done

exit $code
