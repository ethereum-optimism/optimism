#!/bin/bash

set -euo pipefail

SOURCE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
CHALLENGER_DIR=$(echo ${SOURCE_DIR%/*/*})
MONOREPO_DIR=$(echo ${SOURCE_DIR%/*/*/*})

cd $CHALLENGER_DIR
make

cd $MONOREPO_DIR
make devnet-clean
make cannon-prestate
make devnet-up

DEVNET_SPONSOR="ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
DISPUTE_GAME_PROXY=$(jq -r .DisputeGameFactoryProxy $MONOREPO_DIR/.devnet/addresses.json)

echo "----------------------------------------------------------------"
echo " Dispute Game Factory at $DISPUTE_GAME_PROXY"
echo "----------------------------------------------------------------"

L2_OUTPUT_ORACLE_PROXY=$(jq -r .L2OutputOracleProxy $MONOREPO_DIR/.devnet/addresses.json)

echo "----------------------------------------------------------------"
echo " L2 Output Oracle Proxy at $L2_OUTPUT_ORACLE_PROXY"
echo "----------------------------------------------------------------"

BLOCK_ORACLE_PROXY=$(jq -r .BlockOracle $MONOREPO_DIR/.devnet/addresses.json)

echo "----------------------------------------------------------------"
echo " Block Oracle Proxy at $BLOCK_ORACLE_PROXY"
echo "----------------------------------------------------------------"

CHARLIE_ADDRESS="0xF45B7537828CB2fffBC69996B054c2Aaf36DC778"
CHARLIE_KEY="74feb147d72bfae943e6b4e483410933d9e447d5dc47d52432dcc2c1454dabb7"

MALLORY_ADDRESS="0x4641c704a6c743f73ee1f36C7568Fbf4b80681e4"
MALLORY_KEY="28d7045146193f5f4eeb151c4843544b1b0d30a7ac1680c845a416fac65a7715"

echo "----------------------------------------------------------------"
echo " - Fetching balance of the sponsor"
echo " - Balance: $(cast balance 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266)"
echo "----------------------------------------------------------------"

echo "Funding Charlie"
cast send $CHARLIE_ADDRESS --value 5ether --private-key $DEVNET_SPONSOR

echo "Funding Mallory"
cast send $MALLORY_ADDRESS --value 5ether --private-key $DEVNET_SPONSOR

# Loop and wait until there are at least 2 outputs in the l2 output oracle
echo "Waiting until 2 output proposals are in the l2 output oracle..."
echo "NOTE: This may show errors if no output proposals are in the oracle yet."
while [[ $(cast call $L2_OUTPUT_ORACLE_PROXY "latestOutputIndex()" | cast to-dec) -lt 2 ]]
do
  echo "[BLOCK: $(cast block-number)] Waiting for output proposals..."
  sleep 2
done

# Fetch the latest block number
L2_BLOCK_NUMBER=$(cast call $L2_OUTPUT_ORACLE_PROXY "latestBlockNumber()")
echo "Using the latest L2OO block number: $L2_BLOCK_NUMBER"

# We will use the l2 block number of 1 for the dispute game.
# We need to check that the block oracle contains the corresponding l1 block number.
echo "Checkpointing the block oracle..."
L1_CHECKPOINT=$(cast send --private-key $DEVNET_SPONSOR $BLOCK_ORACLE_PROXY "checkpoint()" --json | jq -r .blockNumber | cast to-dec)
((L1_CHECKPOINT=L1_CHECKPOINT-1))
echo "L1 Checkpoint: $L1_CHECKPOINT"

INDEX=$(cast call $L2_OUTPUT_ORACLE_PROXY "getL2OutputIndexAfter(uint256)" $L2_BLOCK_NUMBER | cast to-dec)
((PRIOR_INDEX=INDEX-1))
echo "Getting the l2 output at index $PRIOR_INDEX"
cast call $L2_OUTPUT_ORACLE_PROXY "getL2Output(uint256)" $PRIOR_INDEX

echo "Getting the l2 output at index $INDEX"
cast call $L2_OUTPUT_ORACLE_PROXY "getL2Output(uint256)" $INDEX

# (Alphabet) Fault game type = 0
GAME_TYPE=0

# Root claim commits to the entire trace.
# Alphabet game claim construction: keccak256(abi.encode(trace_index, trace[trace_index]))
ROOT_CLAIM=$(cast keccak $(cast abi-encode "f(uint256,uint256)" 15 122))

# Fault dispute game extra data is calculated as follows.
# abi.encode(uint256(l2_block_number), uint256(l1 checkpoint))
EXTRA_DATA=$(cast abi-encode "f(uint256,uint256)" $L2_BLOCK_NUMBER $L1_CHECKPOINT)

echo "Initializing the game"
FAULT_GAME_ADDRESS=$(cast call --private-key $MALLORY_KEY $DISPUTE_GAME_PROXY "create(uint8,bytes32,bytes)" $GAME_TYPE $ROOT_CLAIM $EXTRA_DATA)

echo "Creating game at address $FAULT_GAME_ADDRESS"
cast send --private-key $MALLORY_KEY $DISPUTE_GAME_PROXY "create(uint8,bytes32,bytes)" $GAME_TYPE $ROOT_CLAIM $EXTRA_DATA

FORMATTED_ADDRESS=$(cast parse-bytes32-address $FAULT_GAME_ADDRESS)
echo "Formatted Address: $FORMATTED_ADDRESS"

echo $FORMATTED_ADDRESS > $CHALLENGER_DIR/.fault-game-address
