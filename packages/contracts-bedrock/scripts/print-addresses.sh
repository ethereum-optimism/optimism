#!/bin/bash

# Get the directory of the script itself
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Check if the first argument is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <directory-name>"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq is not installed. Please install it to run this script."
    exit 1
fi

# Directory name from the argument
DIR_NAME=$1

# Check for the --sdk flag
SDK_MODE=false
if [[ "$2" == "--sdk" ]]; then
    SDK_MODE=true
fi

# Full directory path, relative from the script's location
DIR="$SCRIPT_DIR/../deployments/$DIR_NAME"

# Check if the directory exists
if [ ! -d "$DIR" ]; then
    echo "Directory does not exist: $DIR"
    exit 1
fi

# Declare an array of filenames to check when in SDK mode
declare -a SDK_FILES=("AddressManager" "L1CrossDomainMessengerProxy" "L1StandardBridgeProxy" "OptimismPortalProxy" "L2OutputOracleProxy")

# Loop through each .json file in the directory
for file in "$DIR"/*.json; do
    # Extract the filename without the directory and the .json extension
    filename=$(basename "$file" .json)

    # If SDK mode is on and the filename is not in the list, skip it
    # shellcheck disable=SC2199,SC2076
    if $SDK_MODE && [[ ! " ${SDK_FILES[@]} " =~ " ${filename} " ]]; then
        continue
    fi

    # Extract the 'address' field from the JSON file
    address=$(jq -r '.address' "$file")

    # Print the filename and the address
    echo "${filename}: ${address}"
done
