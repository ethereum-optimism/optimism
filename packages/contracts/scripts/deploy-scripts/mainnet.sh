#!/bin/bash

### All available deploy options at the time of deployment: ###
#  --ctc-force-inclusion-period-seconds  Number of seconds that the sequencer has to include transactions before the L1 queue. (default: 2592000)
#  --ctc-max-transaction-gas-limit       Max gas limit for L1 queue transactions. (default: 11000000)
#  --deploy-scripts                      override deploy script folder path
#  --em-max-gas-per-queue-per-epoch      Maximum gas allowed in a given queue for each epoch. (default: 250000000)
#  --em-max-transaction-gas-limit        Maximum allowed transaction gas limit. (default: 11000000)
#  --em-min-transaction-gas-limit        Minimum allowed transaction gas limit. (default: 50000)
#  --em-ovm-chain-id                     Chain ID for the L2 network. (default: 420)
#  --em-seconds-per-epoch                Number of seconds in each epoch. (default: 0)
#  --export                              export current network deployments
#  --export-all                          export all deployments into one file
#  --gasprice                            gas price to use for transactions
#  --l1-block-time-seconds               Number of seconds on average between every L1 block. (default: 15)
#  --no-compile                          disable pre compilation
#  --no-impersonation                    do not impersonate unknown accounts
#  --ovm-address-manager-owner           Address that will own the Lib_AddressManager. Must be provided or this deployment will fail.
#  --ovm-proposer-address                Address of the account that will propose state roots. Must be provided or this deployment will fail.
#  --ovm-relayer-address                 Address of the message relayer. Must be provided or this deployment will fail.
#  --ovm-sequencer-address               Address of the sequencer. Must be provided or this deployment will fail.
#  --reset                               whether to delete deployments files first
#  --scc-fraud-proof-window              Number of seconds until a transaction is considered finalized. (default: 604800)
#  --scc-sequencer-publish-window        Number of seconds that the sequencer is exclusively allowed to post state roots. (default: 1800)
#  --silent                              whether to remove log
#  --tags                                specify which deploy script to execute via tags, separated by commas
#  --watch                               redeploy on every change of contract or deploy script
#  --write                               whether to write deployments to file


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
 --ctc-force-inclusion-period-seconds 12592000 \
 --ctc-max-transaction-gas-limit 11000000 \
 --em-max-gas-per-queue-per-epoch 250000000 \
 --em-max-transaction-gas-limit 11000000 \
 --em-min-transaction-gas-limit 50000 \
 --em-ovm-chain-id 10 \
 --em-seconds-per-epoch 0 \
 --l1-block-time-seconds 15 \
 --ovm-address-manager-owner 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A \
 --ovm-proposer-address 0x473300df21D047806A082244b417f96b32f13A33 \
 --ovm-relayer-address 0x0000000000000000000000000000000000000000 \
 --ovm-sequencer-address 0x6887246668a3b87F54DeB3b94Ba47a6f63F32985 \
 --reset \
 --scc-fraud-proof-window 604800 \
 --scc-sequencer-publish-window 12592000 \
 --network mainnet

CONTRACTS_TARGET_NETWORK=mainnet \
npx hardhat etherscan-verify --network mainnet
