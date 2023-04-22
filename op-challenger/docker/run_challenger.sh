#! /bin/bash

bin/op-challenger \
    --l1-eth-rpc $OP_CHALLENGER_L1_ETH_RPC \
    --rollup-rpc $OP_CHALLENGER_ROLLUP_RPC \
    --private-key $OP_CHALLENGER_PRIVATE_KEY \
    --dgf-address $OP_CHALLENGER_DGF_ADDRESS \
    --l2oo-address $OP_CHALLENGER_L2OO_ADDRESS
