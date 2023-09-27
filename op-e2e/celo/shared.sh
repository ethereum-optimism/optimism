#!/bin/bash
#shellcheck disable=SC2034  # unused vars make sense in a shared file

export ETH_RPC_URL=http://127.0.0.1:9545

ACC_PRIVKEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
ACC_ADDR=$(cast wallet address $ACC_PRIVKEY)
REGISTRY_ADDR=0x000000000000000000000000000000000000ce10
TOKEN_ADDR=0x471ece3750da237f93b8e339c536989b8978a438
