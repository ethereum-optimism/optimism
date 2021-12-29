#!/bin/bash

### DEPLOYMENT SCRIPT ###
# To be called from root of contracts dir #

ETHERSCAN_API_KEY="1FXFFZ46XQSGCNWFJU7K6ISAGF2EGUKCVP"
CONTRACTS_RPC_URL="https://kovan.infura.io/v3/0c2d27600f8f41c1b37a99b3496f3abb"
CONTRACTS_DEPLOYER_KEY="0x2fc6d7e2abc9120f7cf18b3051af5d6555b64f50b76879c32bf6f0c6b213ed68"

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
#0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244 \
#0x9A2F243c605e6908D96b18e21Fb82Bf288B19EF3 \
#0x18394B52d3Cb931dfA76F63251919D051953413d
CONTRACTS_TARGET_NETWORK=kovan \
npx hardhat deploy \
 --l1-block-time-seconds 15 \
 --ctc-max-transaction-gas-limit 15000000 \
 --ctc-l2-gas-discount-divisor 32 \
 --ctc-enqueue-gas-cost 60000 \
 --scc-fraud-proof-window 10 \
 --scc-sequencer-publish-window 12592000 \
 --ovm-sequencer-address 0xed106C9430594fbB1f97eAA4F04d22E197BfF664 \
 --ovm-proposer-address 0xed106C9430594fbB1f97eAA4F04d22E197BfF664 \
 --ovm-address-manager-owner 0xed106C9430594fbB1f97eAA4F04d22E197BfF664 \
 --gasprice 1000000000 \
 --num-deploy-confirmations 1 \
 --tags upgrade \
 --network kovan

CONTRACTS_TARGET_NETWORK=kovan \
npx hardhat etherscan-verify \
  --network kovan \
  --sleep
