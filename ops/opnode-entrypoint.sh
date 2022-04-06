#!/bin/sh
set -exu

L1_GENESIS=$(curl \
    --fail \
    --retry 10 \
    --retry-delay 2 \
    --retry-connrefused \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}' \
    http://l1:8545)

L2_GENESIS=$(curl \
    --fail \
    --retry 10 \
    --retry-delay 2 \
    --retry-connrefused \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}' \
    http://l2:8545)

echo "L2 genesis timestamp:"
echo $L2_GENESIS | jq -r '.result.timestamp'

cat /rollup.json \
     | jq ". | .genesis.l1.hash = \"$(echo $L1_GENESIS | jq -r '.result.hash')\"" \
     | jq ". | .genesis.l2.hash = \"$(echo $L2_GENESIS | jq -r '.result.hash')\"" \
     | jq ". | .genesis.l2_time = $(echo $L2_GENESIS | jq -r '.result.timestamp' | xargs printf "%d")" \
     | tee /rollup-with-l2-hash.json && \
     mv /rollup-with-l2-hash.json /rollup.json

exec op \
    --l1 ws://l1:8546 \
    --l2 ws://l2:8546 \
    --sequencing.enabled \
    --rollup.config /rollup.json \
    --batchsubmitter.key /config/bss-key.txt \
    --l2.eth http://l2:8545 \
    --rpc.addr 0.0.0.0 \
    --rpc.port 8545
