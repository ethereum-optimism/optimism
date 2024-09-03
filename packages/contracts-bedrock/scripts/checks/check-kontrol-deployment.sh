#!/usr/bin/env bash
set -euo pipefail

# This script checks if the KontrolDeployment.sol file has changed. Removal of
# the DeploymentSummary.t.sol test file means that our primary risk vector for
# KontrolDeployment.sol is an *accidental* change to the file. Changes must
# therefore be explicitly acknowledged by bumping the hash below.

# Get relevant directories
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_BASE=$(dirname "$(dirname "$SCRIPT_DIR")")

# Generate the SHA-512 hash using OpenSSL (very portable)
generated_hash=$(openssl dgst -sha512 "${CONTRACTS_BASE}/test/kontrol/deployment/KontrolDeployment.sol" | awk '{print $2}')

# Define the known hash
known_hash="1664d9c22266b55b43086fa03c0e9d0447b092abc86cba79b86ad36c49167062c2b58a78757a20a5fd257d307599edce8f8f604cc6b2ee86715144015a8c977d"

# Compare the generated hash with the known hash
if [ "$generated_hash" = "$known_hash" ]; then
    echo "KontrolDeployment.sol matches the known hash."
else
    echo "KontrolDeployment.sol does not match the known hash. Please update the known hash."
    echo "Old hash: $known_hash"
    echo "New hash: $generated_hash"
    exit 1
fi
