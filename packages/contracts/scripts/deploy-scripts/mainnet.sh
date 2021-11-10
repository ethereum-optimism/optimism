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

CONTRACTS_TARGET_NETWORK=mainnet \
npx hardhat deploy \
 --l1-block-time-seconds 15 \
 --ctc-max-transaction-gas-limit 15000000 \
 --ctc-l2-gas-discount-divisor 32 \
 --ctc-enqueue-gas-cost 60000 \
 --scc-fraud-proof-window 604800 \
 --scc-sequencer-publish-window 12592000 \
 --ovm-sequencer-address 0x6887246668a3b87F54DeB3b94Ba47a6f63F32985 \
 --ovm-proposer-address 0x473300df21D047806A082244b417f96b32f13A33 \
 --ovm-address-manager-owner 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A \
 --gasprice 150000000000 \
 --num-deploy-confirmations 4 \
 --tags upgrade \
 --network mainnet

CONTRACTS_TARGET_NETWORK=mainnet \
npx hardhat etherscan-verify \
  --network mainnet \
  --sleep
