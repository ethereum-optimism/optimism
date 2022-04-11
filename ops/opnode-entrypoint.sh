#!/bin/sh
set -exu

exec op \
    --l1 ws://l1:8546 \
    --l2 ws://l2:8546 \
    --sequencing.enabled \
    --rollup.config /rollup.json \
    --batchsubmitter.key /config/bss-key.txt \
    --l2.eth http://l2:8545 \
    --rpc.addr 0.0.0.0 \
    --rpc.port 8545
