#!/bin/bash
set -euo pipefail

DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
cd "$DIR"

make

cd ..

make devnet-clean
make devnet-up-deploy

DEVNET_SPONSOR="ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
DISPUTE_GAME_PROXY="0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"

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

# Fault game type = 0
GAME_TYPE=0
# Root claim commits to the entire trace.
# Alphabet game claim construction: keccak256(abi.encode(trace_index, trace[trace_index]))
ROOT_CLAIM=$(cast keccak $(cast abi-encode "f(uint256,uint256)" 15 122))
# Extra data is a dynamic `bytes` type that contains the L2 Block Number of the output proposal that the root claim disagrees with
# Doesn't matter right now since we're not deleting outputs, so just set it to 1
EXTRA_DATA=$(cast to-bytes32 1)

echo "Initializing the game"
cast call --private-key $MALLORY_KEY $DISPUTE_GAME_PROXY "create(uint8,bytes32,bytes)" $GAME_TYPE $ROOT_CLAIM $EXTRA_DATA
cast send --private-key $MALLORY_KEY $DISPUTE_GAME_PROXY "create(uint8,bytes32,bytes)" $GAME_TYPE $ROOT_CLAIM $EXTRA_DATA
