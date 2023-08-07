#!/bin/bash
set -euo pipefail

DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
cd "$DIR"

DISPUTE_GAME_PROXY="0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"

CHARLIE_ADDRESS="0xF45B7537828CB2fffBC69996B054c2Aaf36DC778"
CHARLIE_KEY="74feb147d72bfae943e6b4e483410933d9e447d5dc47d52432dcc2c1454dabb7"

MALLORY_ADDRESS="0x4641c704a6c743f73ee1f36C7568Fbf4b80681e4"
MALLORY_KEY="28d7045146193f5f4eeb151c4843544b1b0d30a7ac1680c845a416fac65a7715"

FAULT_GAME_ADDRESS="0x8daf17a20c9dba35f005b6324f493785d239719d"

PREIMAGE_ORACLE_ADDRESS="0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9"

./bin/op-challenger \
  --l1-eth-rpc http://localhost:8545 \
  --trace-type="alphabet" \
  --alphabet "abcdexyz" \
  --game-address $FAULT_GAME_ADDRESS \
  --preimage-oracle-address $PREIMAGE_ORACLE_ADDRESS \
  --private-key $MALLORY_KEY \
  --num-confirmations 1 \
  --game-depth 4 \
  --agree-with-proposed-output=false
