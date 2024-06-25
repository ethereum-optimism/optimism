#!/bin/bash
#shellcheck disable=SC2034  # unused vars make sense in a shared file

export ETH_RPC_URL=http://127.0.0.1:9545
export ETH_RPC_URL_L1=http://127.0.0.1:8545

export ACC_PRIVKEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
export ACC_ADDR=$(cast wallet address $ACC_PRIVKEY)
export REGISTRY_ADDR=0x000000000000000000000000000000000000ce10
export TOKEN_ADDR=0x471ece3750da237f93b8e339c536989b8978a438
export FEE_CURRENCY_DIRECTORY_ADDR=0x71FFbD48E34bdD5a87c3c683E866dc63b8B2a685
