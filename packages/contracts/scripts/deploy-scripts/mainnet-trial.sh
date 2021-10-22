#!/bin/bash

# ### All available deploy options at the time of deployment: ###
# --ctc-enqueue-gas-cost         	Max gas limit for L1 queue transactions. (default: 60000)
# --ctc-l2-gas-discount-divisor  	Max gas limit for L1 queue transactions. (default: 32)
# --ctc-max-transaction-gas-limit	Max gas limit for L1 queue transactions. (default: 11000000)
# --deploy-scripts               	override deploy script folder path
# --export                       	export current network deployments
# --export-all                   	export all deployments into one file
# --gasprice                     	gas price to use for transactions
# --l1-block-time-seconds        	Number of seconds on average between every L1 block. (default: 15)
# --no-compile                   	disable pre compilation
# --no-impersonation             	do not impersonate unknown accounts
# --num-deploy-confirmations     	Number of confirmations to wait for each transaction in the deployment. More is safer. (default: 12)
# --ovm-address-manager-owner    	Address that will own the Lib_AddressManager. Must be provided or this deployment will fail.
# --ovm-proposer-address         	Address of the account that will propose state roots. Must be provided or this deployment will fail.
# --ovm-sequencer-address        	Address of the sequencer. Must be provided or this deployment will fail.
# --reset                        	whether to delete deployments files first
# --scc-fraud-proof-window       	Number of seconds until a transaction is considered finalized. (default: 604800)
# --scc-sequencer-publish-window 	Number of seconds that the sequencer is exclusively allowed to post state roots. (default: 1800)
# --silent                       	whether to remove log
# --tags                         	specify which deploy script to execute via tags, separated by commas
# --watch                        	redeploy on every change of contract or deploy script
# --write                        	whether to write deployments to file


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
 --ctc-max-transaction-gas-limit 15000000 \
 --ctc-enqueue-gas-cost 60000 \
 --ctc-l2-gas-discount-divisor 32 \
 --l1-block-time-seconds 15 \
 --ovm-address-manager-owner 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A \
 --ovm-sequencer-address 0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244 \
 --ovm-proposer-address 0x9A2F243c605e6908D96b18e21Fb82Bf288B19EF3 \
 --scc-fraud-proof-window 604800 \
 --scc-sequencer-publish-window 12592000 \
 --network mainnet-trial \
 --gasprice 100 \
 --tags upgrade


CONTRACTS_TARGET_NETWORK=mainnet \
npx hardhat etherscan-verify --network mainnet
