#!/bin/sh
set -exu

curl \
    --retry 5 \
    --retry-delay 2 \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}' \
    http://localhost:9545 \
    | jq -r ".result.hash" \
    | tee l2_genesis_hash.txt

cat /rollup.json \
     | jq ". | .genesis.l2.hash = \"${cat l2_genesis_hash.txt}\"" \
     | tee /rollup-with-l2-hash.json && \
     mv /rollup-with-l2-hash.json /rollup.json

exec op \
    --l1 ws://l1:8546 \
    --l2 ws://l2:8546 \
    --sequencing.enabled \
    --rollup.config /config/rollup.json \
    --batchsubmitter.key /config/bss-key.txt \
    --l2.eth http://l2:8545 \
    --rpc.addr 0.0.0.0 \
    --rpc.port 8545
