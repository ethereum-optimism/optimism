#!/usr/bin/env bash
set -euo pipefail

# This script checks for ABI consistency between interfaces and their corresponding contracts.
# It compares the ABIs of interfaces (files starting with 'I') with their implementation contracts,
# excluding constructors and certain predefined files. The script reports any differences found
# and exits with an error if inconsistencies are detected.
# NOTE: Script is fast enough but could be parallelized if necessary.

# Parse flags
no_diff=false
if [[ "${1:-}" == "--no-diff" ]]; then
    no_diff=true
fi

# Grab the directory of the contracts-bedrock package
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_BASE=$(dirname "$(dirname "$SCRIPT_DIR")")

# Define the files to exclude (glob patterns can be used)
EXCLUDE_FILES=(
    # External dependencies
    "IMulticall3"
    "IERC20"
    "IERC721"
    "IERC721Enumerable"
    "IERC721Metadata"
    "IERC721Upgradeable"
    "IERC721Receiver"
    "IERC1271"
    "IERC165"
    "IVotes"
    "IBeacon"
    "IProxyCreationCallback"
    "IAutomate"
    "IGelato1Balance"
    "ERC721TokenReceiver"
    "ERC1155TokenReceiver"
    "ERC777TokensRecipient"
    "Guard"
    "GnosisSafeProxy"

    # Foundry
    "Common"
    "Vm"
    "VmSafe"

    # EAS
    "IEAS"
    "ISchemaResolver"
    "ISchemaRegistry"

    # Kontrol
    "KontrolCheatsBase"
    "KontrolCheats"

    # Error definition files
    "CommonErrors"
    "Errors"

    # TODO: Interfaces that need to be fixed
    "IPreimageOracle"
    "IOptimismMintableERC721"
    "IFaultDisputeGame"
    "IOptimismSuperchainERC20"
    "IInitializable"
    "IOptimismMintableERC20"
    "ILegacyMintableERC20"
    "MintableAndBurnable"
    "IDisputeGameFactory"
    "IWETH"
    "IDelayedWETH"
    "IAnchorStateRegistry"
    "ICrossL2Inbox"
    "IL2ToL2CrossDomainMessenger"
    "KontrolInterfaces"
)

# Convert the exclude files array to a pipe-separated string
EXCLUDE_PATTERN=$( (IFS="|"; echo "${EXCLUDE_FILES[*]}") )

# Find all JSON files in the forge-artifacts folder
JSON_FILES=$(find "$CONTRACTS_BASE/forge-artifacts" -type f -name "*.json" | grep -Ev "$EXCLUDE_PATTERN")

# Initialize a flag to track if any issues are detected
issues_detected=false

# Create a temporary file to store files that have already been reported
REPORTED_INTERFACES_FILE=$(mktemp)

# Define a cleanup function
cleanup() {
    rm -f "$REPORTED_INTERFACES_FILE"
}

# Trap exit and error signals and call cleanup function
trap cleanup EXIT ERR

# Iterate over all JSON files
for interface_file in $JSON_FILES; do
    # Extract contract kind and name in a single pass
    contract_definitions=$(jq -r '.ast.nodes[] | select(.nodeType == "ContractDefinition") | "\(.contractKind),\(.name)"' "$interface_file")

    # Warn and continue if no contract definitions are found
    if [ -z "$contract_definitions" ]; then
        echo "Warning: Could not extract contract definitions from $interface_file."
        echo "Add this file to the EXCLUDE_FILES list if it can be ignored."
        continue
    fi

    while IFS=',' read -r contract_kind contract_name; do
        # If contract kind is not "interface", skip the file
        if [ "$contract_kind" != "interface" ]; then
            continue
        fi

        # If contract name is in the exclude list, skip the file
        # Exclude list functions double duty as a list of files to exclude (glob patterns allowed)
        # and a list of interface names that shouldn't be checked. Simplifies the script a bit and
        # means we can ignore specific interfaces without ignoring the entire file if desired.
        exclude=false
        for exclude_item in "${EXCLUDE_FILES[@]}"; do
            if [[ "$exclude_item" == "$contract_name" ]]; then
                exclude=true
                break
            fi
        done
        if [[ "$exclude" == true ]]; then
            continue
        fi

        # If contract name does not start with an "I", throw an error
        if [[ "$contract_name" != I* ]]; then
            if ! grep -q "^$contract_name$" "$REPORTED_INTERFACES_FILE"; then
                echo "Issue found in ABI for interface $contract_name from file $interface_file."
                echo "Interface $contract_name does not start with 'I'."
                echo "$contract_name" >> "$REPORTED_INTERFACES_FILE"
                issues_detected=true
            fi
            continue
        fi

        # Construct the corresponding contract name by removing the leading "I"
        contract_basename=${contract_name:1}
        corresponding_contract_file="$CONTRACTS_BASE/forge-artifacts/$contract_basename.sol/$contract_basename.json"

        # Check if the corresponding contract file exists
        if [ -f "$corresponding_contract_file" ]; then
            # Extract and compare ABIs excluding constructors
            interface_abi=$(jq '[.abi[] | select(.type != "constructor")]' < "$interface_file")
            contract_abi=$(jq '[.abi[] | select(.type != "constructor")]' < "$corresponding_contract_file")

            # Use jq to compare the ABIs
            if ! diff_result=$(diff -u <(echo "$interface_abi" | jq -S .) <(echo "$contract_abi" | jq -S .)); then
                if ! grep -q "^$contract_name$" "$REPORTED_INTERFACES_FILE"; then
                    echo "Issue found in ABI for interface $contract_name from file $interface_file."
                    echo "Differences found in ABI between interface $contract_name and actual contract $contract_basename."
                    if [ "$no_diff" = false ]; then
                        echo "$diff_result"
                    fi
                    echo "$contract_name" >> "$REPORTED_INTERFACES_FILE"
                    issues_detected=true
                fi
            fi
        fi
    done <<< "$contract_definitions"
done

# Fail the script if any issues were detected
if [ "$issues_detected" = true ]; then
    echo "Issues were detected while validating interface files."
    echo "If the interface is an external dependency or should otherwise be excluded from this"
    echo "check, add the interface name to the EXCLUDE_FILES list in the script. This will prevent"
    echo "the script from comparing it against a corresponding contract."
    exit 1
else
    exit 0
fi
