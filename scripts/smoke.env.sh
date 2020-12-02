#!/bin/sh

export MNEMONIC="explain foam nice clown method avocado hill basket echo blur elevator marble"
export CHAIN_ID=5777
export PORT=8545
export RPC_URL="http://ganache:$PORT"
export CONTRACTS_PATH="/vault/contracts/erc20/build/"
export PLASMA_CONTRACT=`cat /truffleshuffle/plasma_framework_addr.out`
export GAS_PRICE_LOW="1"
export GAS_PRICE_HIGH="37000000000"
