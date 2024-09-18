#!/usr/bin/env bash
set -euo pipefail

# Grab the directory of the contracts-bedrock package
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_BASE=$(dirname "$(dirname "$SCRIPT_DIR")")
ARTIFACTS_DIR="$CONTRACTS_BASE/forge-artifacts"
CONTRACTS_DIR="$CONTRACTS_BASE/src"

# Load semver-utils
# shellcheck source=/dev/null
source "$SCRIPT_DIR/utils/semver-utils.sh"

# Flag to track if any errors are detected
has_errors=false

# Iterate through each artifact file
for artifact_file in "$ARTIFACTS_DIR"/**/*.json; do
    # Get the contract name and find the corresponding source file
    contract_name=$(basename "$artifact_file" .json)
    contract_file=$(find "$CONTRACTS_DIR" -name "$contract_name.sol")

    # Try to extract version as a constant
    raw_metadata=$(jq -r '.rawMetadata' "$artifact_file")
    artifact_version=$(echo "$raw_metadata" | jq -r '.output.devdoc.stateVariables.version."custom:semver"')

    is_constant=true
    if [ "$artifact_version" = "null" ]; then
        # If not found as a constant, try to extract as a function
        artifact_version=$(echo "$raw_metadata" | jq -r '.output.devdoc.methods."version()"."custom:semver"')
        is_constant=false
    fi

    # If @custom:semver is not found in either location, skip this file
    if [ "$artifact_version" = "null" ]; then
        continue
    fi

    # If source file is not found, report an error
    if [ -z "$contract_file" ]; then
        echo "❌ $contract_name: Source file not found"
        continue
    fi

    # Extract version from source based on whether it's a constant or function
    if [ "$is_constant" = true ]; then
        source_version=$(extract_constant_version "$contract_file")
    else
        source_version=$(extract_function_version "$contract_file")
    fi

    # If source version is not found, report an error
    if [ "$source_version" = "" ]; then
        echo "❌ Error: failed to find version string for $contract_name"
        echo "        this is probably a bug in check-contract-semver.sh"
        echo "        please report or fix the issue if possible"
        has_errors=true
    fi

    # Compare versions
    if [ "$source_version" != "$artifact_version" ]; then
        echo "❌ Error: $contract_name has different semver in code and devdoc"
        echo "   Code: $source_version"
        echo "   Devdoc: $artifact_version"
        has_errors=true
    else
        echo "✅ $contract_name: code: $source_version, devdoc: $artifact_version"
    fi
done

# If any errors were detected, exit with a non-zero status
if [ "$has_errors" = true ]; then
    exit 1
fi
