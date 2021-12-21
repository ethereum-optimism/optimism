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

DEPLOY_CMD="npx hardhat deploy"

# Helper method to concatenate things onto $DEPLOY_CMD.
# Usage: concat_cmd "str-to-concat"
function concat_cmd() {
    DEPLOY_CMD="$DEPLOY_CMD $1"
}

# Helper method to conditionally concatenate CLI arguments
# when a given env var is defined.
# Usage: concat_arg "--arg-name" "env-var-name"
function concat_arg() {
    ARG=$1
    ENV_VAR=$2
    if [ -n "${!ENV_VAR+x}" ]; then
        echo "$ENV_VAR set to ${!ENV_VAR}, applying $ARG argument."
        concat_cmd "$ARG \"${!ENV_VAR}\""
    else
        echo "$ENV_VAR is not not set, skipping $ARG argument."
    fi
}

concat_arg "--network" "CONTRACTS_TARGET_NETWORK"
concat_arg "--ovm-address-manager-owner" "OVM_ADDRESS_MANAGER_OWNER"
concat_arg "--ovm-proposer-address" "OVM_PROPOSER_ADDRESS"
concat_arg "--ovm-sequencer-address" "OVM_SEQUENCER_ADDRESS"
concat_arg "--l1-block-time-seconds" "L1_BLOCK_TIME_SECONDS"
concat_arg "--ctc-max-transaction-gas-limit" "CTC_MAX_TRANSACTION_GAS_LIMIT"
concat_arg "--ctc-l2-gas-discount-divisor" "CTC_L2_GAS_DISCOUNT_DIVISOR"
concat_arg "--ctc-enqueue-gas-cost" "CTC_ENQUEUE_GAS_COST"
concat_arg "--scc-fraud-proof-window" "SCC_FRAUD_PROOF_WINDOW"
concat_arg "--num-deploy-confirmations" "NUM_DEPLOY_CONFIRMATIONS"
concat_arg "--forked" "FORKED"

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

echo "Env vars for the dump script:"
export L1_STANDARD_BRIDGE_ADDRESS=$(cat "./addresses.json" | jq -r .Proxy__OVM_L1StandardBridge)
export L1_CROSS_DOMAIN_MESSENGER_ADDRESS=$(cat "./addresses.json" | jq -r .Proxy__OVM_L1CrossDomainMessenger)
echo "ADDRESS_MANAGER_ADDRESS=$ADDRESS_MANAGER_ADDRESS"
echo "L1_STANDARD_BRIDGE_ADDRESS=$L1_STANDARD_BRIDGE_ADDRESS"
echo "L1_CROSS_DOMAIN_MESSENGER_ADDRESS=$L1_CROSS_DOMAIN_MESSENGER_ADDRESS"


echo "Building dump file."
yarn run build:dump
mv addresses.json ./dist/dumps

# service the addresses and dumps
echo "Starting server."
python3 -m http.server \
    --bind "0.0.0.0" 8081 \
    --directory ./dist/dumps
