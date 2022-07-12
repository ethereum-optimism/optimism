#!/usr/bin/env bash

# This script starts the integration tests using the deployed contracts. The script
# takes one argument: a path to the deployments directory.

set -eu

DEPLOYMENTS_DIR="$1"

if [ ! -f "$DEPLOYMENTS_DIR/OptimismPortal.json" ]; then
  echo "Deployment directory $DEPLOYMENTS_DIR not found. Please "
  echo "check the path, then try again."
fi

export OPTIMISM_PORTAL_ADDRESS=$(jq -r '.address' < "$DEPLOYMENTS_DIR/OptimismPortal.json")
cd ./packages/integration-tests-bedrock
yarn test
