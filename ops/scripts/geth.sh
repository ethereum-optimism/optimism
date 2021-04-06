#!/bin/bash
RETRIES=${RETRIES:-20}
# get the addrs from the URL provided
ADDRESSES=$(curl --retry-connrefused --retry $RETRIES --retry-delay 1 $URL)

function envSet() {
    VAR=$1
    export $VAR=$(echo $ADDRESSES | jq -r ".$2")
}

# set all the necessary env vars
envSet ETH1_ADDRESS_RESOLVER_ADDRESS  AddressManager
envSet ETH1_L1_CROSS_DOMAIN_MESSENGER_ADDRESS Proxy__OVM_L1CrossDomainMessenger
envSet ETH1_L1_ETH_GATEWAY_ADDRESS OVM_L1ETHGateway
envSet ROLLUP_ADDRESS_MANAGER_OWNER_ADDRESS Deployer

geth --verbosity=6
