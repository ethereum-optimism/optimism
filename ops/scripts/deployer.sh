#!/bin/bash
set -euo

RETRIES=${RETRIES:-20}
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

if [ -z "$CONTRACTS_RPC_URL" ]; then
    echo "Must specify \$CONTRACTS_RPC_URL."
    exit 1
fi

# wait for the base layer to be up
curl \
    --fail \
    --show-error \
    --silent \
    -H "Content-Type: application/json" \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    -d $JSON \
    $CONTRACTS_RPC_URL > /dev/null

echo "Connected to L1."
echo "Building deployment command."

DEPLOY_CMD="npx hardhat deploy --network $CONTRACTS_TARGET_NETWORK"

echo "Deploying contracts. Deployment command:"
echo "$DEPLOY_CMD"
eval "$DEPLOY_CMD"

echo "Building addresses.json."
export ADDRESS_MANAGER_ADDRESS=$(cat "./deployments/$CONTRACTS_TARGET_NETWORK/Lib_AddressManager.json" | jq -r .address)

# First, create two files. One of them contains a list of addresses, the other contains a list of contract names.
find "./deployments/$CONTRACTS_TARGET_NETWORK" -maxdepth 1 -name '*.json' | xargs cat | jq -r '.address' > addresses.txt
find "./deployments/$CONTRACTS_TARGET_NETWORK" -maxdepth 1 -name '*.json' | sed -e "s/.\/deployments\/$CONTRACTS_TARGET_NETWORK\///g" | sed -e 's/.json//g' > filenames.txt

# Start building addresses.json.
echo "{" >> addresses.json
# Zip the two files describe above together, then, switch their order and format as JSON.
paste addresses.txt filenames.txt | sed -e "s/^\([^ ]\+\)\s\+\([^ ]\+\)/\"\2\": \"\1\",/" >> addresses.json
# Add the address manager alias.
echo "\"AddressManager\": \"$ADDRESS_MANAGER_ADDRESS\"" >> addresses.json
# End addresses.json
echo "}" >> addresses.json

echo "Built addresses.json. Content:"
jq . addresses.json

echo "Building dump file."
npx hardhat take-dump --network $CONTRACTS_TARGET_NETWORK
mv addresses.json ./genesis
cp ./genesis/$CONTRACTS_TARGET_NETWORK.json ./genesis/state-dump.latest.json

# expose the deployments
cp -rf ./deployments ./genesis/deployments

# service the addresses and dumps
echo "Starting server."
python3 -m http.server \
    --bind "0.0.0.0" 8081 \
    --directory ./genesis
