#!/bin/bash

### DEPLOYMENT SCRIPT ###
# To be called from root of contracts dir #

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

CONTRACTS_TARGET_NETWORK=goerli \
npx hardhat deploy \
 --l1-block-time-seconds 15 \
 --ctc-max-transaction-gas-limit 15000000 \
 --ctc-l2-gas-discount-divisor 32 \
 --ctc-enqueue-gas-cost 60000 \
 --scc-fraud-proof-window 604800 \
 --scc-sequencer-publish-window 12592000 \
 --ovm-sequencer-address 0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244 \
 --ovm-proposer-address 0x9A2F243c605e6908D96b18e21Fb82Bf288B19EF3 \
 --ovm-address-manager-owner 0x32b70c156302d28A9119445d2bbb9ab1cBD01671 \
 --network goerli

CONTRACTS_TARGET_NETWORK=goerli \
npx hardhat etherscan-verify --network goerli
