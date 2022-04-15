#!/usr/bin/env bash

# This script starts a local devnet using Docker Compose. We have to use
# this more complicated Bash script rather than Compose's native orchestration
# tooling because we need to start each service in a specific order, and specify
# their configuration along the way. The order is:
#
# 1. Start L1.
# 2. Compile contracts.
# 3. Deploy the contracts to L1 if necessary.
# 4. Start L2, inserting the compiled contract artifacts into the genesis.
# 5. Get the genesis hashes and timestamps from L1/L2.
# 6. Generate the rollup driver's config using the genesis hashes and the
#    timestamps recovered in step 4 as well as the address of the OptimismPortal
#    contract deployed in step 3.
# 7. Start the rollup driver.
# 8. Start the L2 output submitter.
#
# The timestamps are critically important here, since the rollup driver will fill in
# empty blocks if the tip of L1 lags behind the current timestamp. This can lead to
# a perceived infinite loop. To get around this, we set the timestamp to the current
# time in this script.
#
# This script is safe to run multiple times. It stores state in `.devnet`, and
# packages/contracts/deployments/devnetL1.
#
# Don't run this script directly. Run it using the makefile, e.g. `make devnet-up`.
# To clean up your devnet, run `make devnet-clean`.

set -eu

L1_URL="http://localhost:8545"
L2_URL="http://localhost:9545"

# Helper method that waits for a given URL to be up. Can't use
# cURL's built-in retry logic because connection reset errors
# are ignored unless you're using a very recent version of cURL
function wait_up {
  echo -n "Waiting for $1 to come up..."
  i=0
  until curl -s -f -o /dev/null "$1"
  do
    echo -n .
    sleep 0.25

    ((i=i+1))
    if [ "$i" -eq 120 ]; then
      echo " Timeout!" >&2
      exit 0
    fi
  done
  echo "Done!"
}

# Regenerate the L1 genesis file if necessary. The existence of the genesis
# file is used to determine if we need to recreate the devnet's state folder.
if [ ! -f ./.devnet/genesis-l1.json ]; then
  echo "Regenerating L1 genesis."
  mkdir -p ./.devnet
  GENESIS_TIMESTAMP=$(date +%s | xargs printf "0x%x")
  jq ". | .timestamp = \"$GENESIS_TIMESTAMP\" " < ./ops/genesis-l1.json > ./.devnet/genesis-l1.json
else
  GENESIS_TIMESTAMP=$(jq -r '.timestamp' < ./.devnet/genesis-l1.json)
fi

# Bring up L1.
cd ops
echo "Bringing up L1..."
DOCKER_BUILDKIT=1 docker-compose build
docker-compose up -d l1
wait_up $L1_URL
cd ../

# Deploy contracts using Hardhat.
if [ ! -f ./packages/contracts/deployments/devnetL1/OptimismPortal.json ]; then
  echo "Deploying contracts."
  cd ./packages/contracts
  L2OO_STARTING_BLOCK_TIMESTAMP=$GENESIS_TIMESTAMP yarn hardhat --network devnetL1 deploy
  cd ../../
else
  echo "Contracts already deployed, skipping."
fi

# Pull out the necessary bytecode/addresses from the artifacts/deployments.
WITHDRAWER_BYTECODE=$(jq -r .deployedBytecode < ./packages/contracts/artifacts/contracts/L2/Withdrawer.sol/Withdrawer.json)
L1_BLOCK_INFO_BYTECODE=$(jq -r .deployedBytecode < ./packages/contracts/artifacts/contracts/L2/L1Block.sol/L1Block.json)
DEPOSIT_CONTRACT_ADDRESS=$(jq -r .address < ./packages/contracts/deployments/devnetL1/OptimismPortal.json)
L2OO_ADDRESS=$(jq -r .address < ./packages/contracts/deployments/devnetL1/L2OutputOracle.json)

# Replace values in the L2 genesis file. It doesn't matter if this gets run every time,
# since the replaced values will be the same.
jq ". | .alloc.\"4200000000000000000000000000000000000015\".code = \"$L1_BLOCK_INFO_BYTECODE\"" < ./ops/genesis-l2.json | \
  jq ". | .alloc.\"4200000000000000000000000000000000000015\".balance = \"0x0\"" | \
  jq ". | .alloc.\"4200000000000000000000000000000000000016\".code = \"$WITHDRAWER_BYTECODE\"" | \
  jq ". | .alloc.\"4200000000000000000000000000000000000016\".balance = \"0x0\"" | \
  jq ". | .timestamp = \"$GENESIS_TIMESTAMP\" " > ./.devnet/genesis-l2.json

# Bring up L2.
cd ops
echo "Bringing up L2..."
docker-compose up -d l2
wait_up $L2_URL
cd ../

# Start putting together the rollup config.
echo "Building rollup config..."

# Grab the L1 genesis. We can use cURL here to retry.
L1_GENESIS=$(curl \
    --silent \
    --fail \
    --retry 10 \
    --retry-delay 2 \
    --retry-connrefused \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}' \
    $L1_URL)

# Grab the L2 genesis. We can use cURL here to retry.
L2_GENESIS=$(curl \
    --silent \
    --fail \
    --retry 10 \
    --retry-delay 2 \
    --retry-connrefused \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}' \
    $L2_URL)

# Generate the rollup config.
jq ". | .genesis.l1.hash = \"$(echo $L1_GENESIS | jq -r '.result.hash')\"" < ./ops/rollup.json | \
   jq ". | .genesis.l2.hash = \"$(echo $L2_GENESIS | jq -r '.result.hash')\"" | \
   jq ". | .genesis.l2_time = $(echo $L2_GENESIS | jq -r '.result.timestamp' | xargs printf "%d")" | \
   jq ". | .deposit_contract_address = \"$DEPOSIT_CONTRACT_ADDRESS\"" > ./.devnet/rollup.json


SEQUENCER_GENESIS_HASH="$(echo $L2_GENESIS | jq -r '.result.hash')"
SEQUENCER_BATCH_INBOX_ADDRESS="$(cat ./ops/rollup.json | jq -r '.batch_inbox_address')"

# Bring up everything else.
cd ops
echo "Bringing up devnet..."
L2OO_ADDRESS="$L2OO_ADDRESS" \
	SEQUENCER_GENESIS_HASH="$SEQUENCER_GENESIS_HASH" \
	SEQUENCER_BATCH_INBOX_ADDRESS="$SEQUENCER_BATCH_INBOX_ADDRESS" \
	docker-compose up -d l2os bss
cd ../

echo "Devnet ready."
