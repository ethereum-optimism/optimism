#!/bin/bash

export DATA_TRANSPORT_LAYER__ADDRESS_MANAGER='0x18394b52d3cb931dfa76f63251919d051953413d'

npx lerna exec --scope @eth-optimism/data-transport-layer --no-bail yarn start -- \
    --db-path /mnt/2tb-ssd/kovan-regenesis/dtl \
    --confirmations 12 \
    --dangerously-catch-all-errors true \
    --sync-from-l1 false \
    --sync-from-l2 true \
    --l2-rpc-endpoint https://kovan.optimism.io \
    --l2-chainid 69 \
    --default-backend l2
