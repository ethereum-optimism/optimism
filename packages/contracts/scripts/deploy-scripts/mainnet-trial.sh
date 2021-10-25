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

CONTRACTS_TARGET_NETWORK=mainnet-trial \
npx hardhat deploy \
  --ctc-max-transaction-gas-limit 15000000 \
  --ctc-enqueue-gas-cost 60000 \
  --ctc-l2-gas-discount-divisor 32 \
  --l1-block-time-seconds 15 \
  --ovm-address-manager-owner 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A \
  --ovm-sequencer-address 0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244 \
  --ovm-proposer-address 0x9A2F243c605e6908D96b18e21Fb82Bf288B19EF3 \
  --scc-fraud-proof-window 10 \
  --scc-sequencer-publish-window 12592000 \
  --network mainnet-trial \
  --gasprice 2000000000000 \
  --forked true \
  --num-deploy-confirmations 0 \
  --tags upgrade
