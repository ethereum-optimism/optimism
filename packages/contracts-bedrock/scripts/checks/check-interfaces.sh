#!/usr/bin/env bash
set -euo pipefail

# This script checks for ABI consistency between interfaces and their corresponding contracts.
# It compares the ABIs of interfaces (files starting with 'I') with their implementation contracts,
# excluding certain predefined files. Constructors are expected to be represented in interfaces by a
# pseudo-constructor function `__constructor__(...)` with arguments the same as the contract's constructor.
# The script reports any differences found and exits with an error if inconsistencies are detected.
# NOTE: Script is fast enough but could be parallelized if necessary.

# Parse flags
no_diff=false
if [[ "${1:-}" == "--no-diff" ]]; then
    no_diff=true
fi

# Grab the directory of the contracts-bedrock package
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_BASE=$(dirname "$(dirname "$SCRIPT_DIR")")

# Define contracts that should be excluded from the check
EXCLUDE_CONTRACTS=(
    # External dependencies
    "IERC20"
    "IERC721"
    "IERC721Enumerable"
    "IERC721Upgradeable"
    "IERC721Metadata"
    "IERC165"
    "IERC165Upgradeable"
    "ERC721TokenReceiver"
    "ERC1155TokenReceiver"
    "ERC777TokensRecipient"
    "Guard"
    "IProxy"
    "Vm"
    "VmSafe"
    "IMulticall3"
    "IERC721TokenReceiver"
    "IProxyCreationCallback"
    "IBeacon"

    # EAS
    "IEAS"
    "ISchemaResolver"
    "ISchemaRegistry"

    # Kontrol
    "KontrolCheatsBase"

    # TODO: Interfaces that need to be fixed
    "IOptimismSuperchainERC20"
    "IOptimismMintableERC721"
    "IOptimismMintableERC20"
    "ILegacyMintableERC20"
    "IInitializable"
    "IPreimageOracle"
    "ICrossL2Inbox"
    "IL2ToL2CrossDomainMessenger"
    "MintableAndBurnable"
    "IWETH"
    "IDelayedWETH"
    "IResolvedDelegateProxy"

    # TODO: Kontrol interfaces that need to be removed
    "IL1ERC721Bridge"
    "IL1StandardBridge"
    "IL1CrossDomainMessenger"
    "ISuperchainConfig"
    "IOptimismPortal"
)

# Find all JSON files in the forge-artifacts folder
JSON_FILES=$(find "$CONTRACTS_BASE/forge-artifacts" -type f -name "*.json")

# Initialize a flag to track if any issues are detected
issues_detected=false

# Create a temporary file to store files that have already been reported
REPORTED_INTERFACES_FILE=$(mktemp)

# Clean up the temporary file on exit
cleanup() {
    rm -f "$REPORTED_INTERFACES_FILE"
}

# Trap exit and error signals and call cleanup function
trap cleanup EXIT ERR

# Check if a contract is excluded
is_excluded() {
    for exclude in "${EXCLUDE_CONTRACTS[@]}"; do
        if [[ "$exclude" == "$1" ]]; then
            return 0
        fi
    done
    return 1
}

# Iterate over all JSON files
for interface_file in $JSON_FILES; do
    # Grab the contract name from the file name
    contract_name=$(basename "$interface_file" .json | cut -d '.' -f 1)

    # Extract all contract definitions in a single pass
    contract_definitions=$(jq -r '.ast.nodes[] | select(.nodeType == "ContractDefinition") | "\(.contractKind),\(.name)"' "$interface_file")

    # Continue if no contract definitions are found
    # Can happen in Solidity files that don't declare any contracts/interfaces
    if [ -z "$contract_definitions" ]; then
        continue
    fi

    # Iterate over the found contract definitions and figure out which one
    # matches the file name. We do this so that we can figure out if this is an
    # interface or a contract based on the contract kind.
    found=false
    contract_temp=""
    contract_kind=""
    for definition in $contract_definitions; do
        IFS=',' read -r contract_kind contract_temp <<< "$definition"
        if [[ "$contract_name" == "$contract_temp" ]]; then
            found=true
            break
        fi
    done

    # Continue if no matching contract name is found. Can happen in Solidity
    # files where no contracts/interfaces are defined with the same name as the
    # file. Still OK because a separate artifact *will* be generated for the
    # specific contract/interface.
    if [ "$found" = false ]; then
        continue
    fi

    # If contract kind is not "interface", skip the file
    if [ "$contract_kind" != "interface" ]; then
        continue
    fi

    # If contract name does not start with an "I", throw an error
    if [[ "$contract_name" != I* ]]; then
        if ! grep -q "^$contract_name$" "$REPORTED_INTERFACES_FILE"; then
            echo "$contract_name" >> "$REPORTED_INTERFACES_FILE"
            if ! is_excluded "$contract_name"; then
                echo "Issue found in ABI for interface $contract_name from file $interface_file."
                echo "Interface $contract_name does not start with 'I'."
                issues_detected=true
            fi
        fi
        continue
    fi

    # Extract contract semver
    contract_semver=$(jq -r '.ast.nodes[] | select(.nodeType == "PragmaDirective") | .literals | join("")' "$interface_file")

    # If semver is not exactly "solidity^0.8.0", throw an error
    if [ "$contract_semver" != "solidity^0.8.0" ]; then
        if ! grep -q "^$contract_name$" "$REPORTED_INTERFACES_FILE"; then
            echo "$contract_name" >> "$REPORTED_INTERFACES_FILE"
            if ! is_excluded "$contract_name"; then
                echo "Issue found in ABI for interface $contract_name from file $interface_file."
                echo "Interface $contract_name does not have correct compiler version (MUST be exactly solidity ^0.8.0)."
                issues_detected=true
            fi
        fi
        continue
    fi

    # Construct the corresponding contract name by removing the leading "I"
    contract_basename=${contract_name:1}
    corresponding_contract_file="$CONTRACTS_BASE/forge-artifacts/$contract_basename.sol/$contract_basename.json"

    # Skip the file if the corresponding contract file does not exist
    if [ ! -f "$corresponding_contract_file" ]; then
        continue
    fi

    # Extract and compare ABIs excluding constructors
    interface_abi=$(jq '[.abi[]]' < "$interface_file")
    contract_abi=$(jq '[.abi[]]' < "$corresponding_contract_file")

    # Function to normalize ABI by replacing interface name with contract name.
    # Base contracts aren't allowed to inherit from their interfaces in order
    # to guarantee a 1:1 match between interfaces and contracts. This means
    # that the interface will redefine types in the base contract. We normalize
    # the ABI as if the interface and contract are the same name.
    normalize_abi() {
        # Here we just remove the leading "I" from any contract, enum, or
        # struct type. It's not beautiful but it's good enough for now. It
        # would miss certain edge cases like if an interface really is using
        # the contract type instead of the interface type but that's unlikely
        # to happen in practice and should be an easy fix if it does.
        local abi="$1"

        # Remove the leading "I" from types.
        abi="${abi//\"internalType\": \"contract I/\"internalType\": \"contract }"
        abi="${abi//\"internalType\": \"enum I/\"internalType\": \"enum }"
        abi="${abi//\"internalType\": \"struct I/\"internalType\": \"struct }"

        # Handle translating pseudo-constructors.
        abi=$(echo "$abi" | jq 'map(if .type == "function" and .name == "__constructor__" then .type = "constructor" | del(.name) | del(.outputs) else . end)')

        echo "$abi"
    }

    # Normalize the ABIs
    normalized_interface_abi=$(normalize_abi "$interface_abi")
    normalized_contract_abi=$(normalize_abi "$contract_abi")

    # Use jq to compare the ABIs
    if ! diff_result=$(diff -u <(echo "$normalized_interface_abi" | jq 'sort') <(echo "$normalized_contract_abi" | jq 'sort')); then
        if ! grep -q "^$contract_name$" "$REPORTED_INTERFACES_FILE"; then
            echo "$contract_name" >> "$REPORTED_INTERFACES_FILE"
            if ! is_excluded "$contract_name"; then
                echo "Issue found in ABI for interface $contract_name from file $interface_file."
                echo "Differences found in ABI between interface $contract_name and actual contract $contract_basename."
                if [ "$no_diff" = false ]; then
                    echo "$diff_result"
                fi
                issues_detected=true
            fi
        fi
        continue
    fi
done

# Check for unnecessary exclusions
for exclude_item in "${EXCLUDE_CONTRACTS[@]}"; do
    if ! grep -q "^$exclude_item$" "$REPORTED_INTERFACES_FILE"; then
        echo "Warning: $exclude_item is in the exclude list but WAS NOT reported as an issue. It"
        echo "may be unnecessary in the EXCLUDE_CONTRACTS list, but you MUST verify this before"
        echo "removing it by performing a clean and full build before re-running this script."
    fi
done

# Fail the script if any issues were detected
if [ "$issues_detected" = true ]; then
    echo "Issues were detected while validating interface files."
    echo "If the interface is an external dependency or should otherwise be excluded from this"
    echo "check, add the interface name to the EXCLUDE_CONTRACTS list in the script. This will prevent"
    echo "the script from comparing it against a corresponding contract."
    echo "IMPORTANT: Interface files are NOT yet generated automatically. You must fix any"
    echo "listed discrepancies manually by updating the specified interface file. Automated"
    echo "interface generation is dependent on a few Forge bug fixes."
    exit 1
else
    exit 0
fi
