#!/bin/bash

# Required env vars
if [[ -z "$CONTRACTS_DEPLOYER_KEY" ]]; then
  echo "Must pass CONTRACTS_DEPLOYER_KEY"
  exit 1
fi
if [[ -z "$CONTRACTS_RPC_URL" ]]; then
  echo "Must pass CONTRACTS_RPC_URL"
  exit 1
fi
if [[ -z "$ETHERSCAN_API_KEY" ]]; then
  echo "Must pass ETHERSCAN_API_KEY"
  exit 1
fi

# TODO addresses

CONTRACTS_TARGET_NETWORK=mainnet \
npx hardhat deploy \
 --l1-block-time-seconds 15 \
 --ctc-max-transaction-gas-limit 15000000 \
 --ctc-l2-gas-discount-divisor 32 \
 --ctc-enqueue-gas-cost 60000 \
 --scc-fraud-proof-window 604800 \
 --scc-sequencer-publish-window 12592000 \
 --ovm-address-manager-owner 0x \
 --ovm-proposer-address 0x \
 --ovm-sequencer-address 0x \
 --tags upgrade \
 --network mainnet

CONTRACTS_TARGET_NETWORK=mainnet \
npx hardhat etherscan-verify --network mainnet
