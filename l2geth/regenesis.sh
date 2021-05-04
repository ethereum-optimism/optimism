#!/bin/bash

./scripts/start.sh \
    --datadir /mnt/2tb-ssd/kovan-regenesis/geth \
    --eth1.ctcdeploymentheight 24199483 \
    --chainid 69 \
    --eth1.l1crossdomainmessengeraddress 0x48062eD9b6488EC41c4CfbF2f568D7773819d8C9 \
    --rollup.addressmanagerowneraddress 0x18394b52d3cb931dfa76f63251919d051953413d \
    --eth1.l1gatewayaddress 0xf3902e50dA095bD2e954AB320E8eafDA6152dFDa \
    --rollup.statedumppath https://storage.googleapis.com/optimism/kovan/3.json \
    --verifier
