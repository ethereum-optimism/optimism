#!/bin/sh

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

export CHAIN_ID=`/opt/secret2env -name $SECRETNAME|grep -w CHAIN_ID|sed 's/CHAIN_ID=//g'`
export DATADIR=`/opt/secret2env -name $SECRETNAME|grep -w DATADIR|sed 's/DATADIR=//g'`
export DEV=`/opt/secret2env -name $SECRETNAME|grep -w DEV|sed 's/DEV=//g'`
export ETH1_CONFIRMATION_DEPTH=`/opt/secret2env -name $SECRETNAME|grep -w ETH1_CONFIRMATION_DEPTH|sed 's/ETH1_CONFIRMATION_DEPTH=//g'`
export ETH1_CTC_DEPLOYMENT_HEIGHT=`/opt/secret2env -name $SECRETNAME|grep -w ETH1_CTC_DEPLOYMENT_HEIGHT|sed 's/ETH1_CTC_DEPLOYMENT_HEIGHT=//g'`
export ETH1_SYNC_SERVICE_ENABLE=`/opt/secret2env -name $SECRETNAME|grep -w ETH1_SYNC_SERVICE_ENABLE|sed 's/ETH1_SYNC_SERVICE_ENABLE=//g'`
export GASPRICE=`/opt/secret2env -name $SECRETNAME|grep -w GASPRICE|sed 's/GASPRICE=//g'`
export GCMODE=`/opt/secret2env -name $SECRETNAME|grep -w GCMODE|sed 's/GCMODE=//g'`
export IPC_DISABLE=`/opt/secret2env -name $SECRETNAME|grep -w IPC_DISABLE|sed 's/IPC_DISABLE=//g'`
export NETWORK_ID=`/opt/secret2env -name $SECRETNAME|grep -w NETWORK_ID|sed 's/NETWORK_ID=//g'`
export NO_DISCOVER=`/opt/secret2env -name $SECRETNAME|grep -w NO_DISCOVER|sed 's/NO_DISCOVER=//g'`
export NO_USB=`/opt/secret2env -name $SECRETNAME|grep -w NO_USB|sed 's/NO_USB=//g'`
export ROLLUP_POLL_INTERVAL_FLAG=`/opt/secret2env -name $SECRETNAME|grep -w ROLLUP_POLL_INTERVAL_FLAG|sed 's/ROLLUP_POLL_INTERVAL_FLAG=//g'`
export RPC_API=`/opt/secret2env -name $SECRETNAME|grep -w RPC_API|sed 's/RPC_API=//g'`
export RPC_CORS_DOMAIN=`/opt/secret2env -name $SECRETNAME|grep -w RPC_CORS_DOMAIN|sed 's/RPC_CORS_DOMAIN=//g'`
export RPC_ENABLE=`/opt/secret2env -name $SECRETNAME|grep -w RPC_ENABLE|sed 's/RPC_ENABLE=//g'`
export RPC_PORT=`/opt/secret2env -name $SECRETNAME|grep -w RPC_PORT|sed 's/RPC_PORT=//g'`
export RPC_VHOSTS=`/opt/secret2env -name $SECRETNAME|grep -w RPC_VHOSTS|sed 's/RPC_VHOSTS=//g'`
export TARGET_GAS_LIMIT=`/opt/secret2env -name $SECRETNAME|grep -w TARGET_GAS_LIMIT|sed 's/TARGET_GAS_LIMIT=//g'`
export USING_OVM=`/opt/secret2env -name $SECRETNAME|grep -w USING_OVM|sed 's/USING_OVM=//g'`
export WS=`/opt/secret2env -name $SECRETNAME|grep -w WS|sed 's/WS=//g'`
export WS_ADDR=`/opt/secret2env -name $SECRETNAME|grep -w WS_ADDR|sed 's/WS_ADDR=//g'`
export WS_API=`/opt/secret2env -name $SECRETNAME|grep -w WS_API|sed 's/WS_API=//g'`
export WS_ORIGINS=`/opt/secret2env -name $SECRETNAME|grep -w WS_ORIGINS|sed 's/WS_ORIGINS=//g'`

RETRIES=${RETRIES:-40}
VERBOSITY=${VERBOSITY:-6}

if [[ ! -z "$URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)

    function envSet() {
        VAR=$1
        export $VAR=$(echo $ADDRESSES | jq -r ".$2")
    }

    # set all the necessary env vars
    envSet ETH1_ADDRESS_RESOLVER_ADDRESS  AddressManager
    envSet ETH1_L1_CROSS_DOMAIN_MESSENGER_ADDRESS Proxy__OVM_L1CrossDomainMessenger
    envSet ROLLUP_ADDRESS_MANAGER_OWNER_ADDRESS Deployer

    # set the address to the proxy gateway if possible
    envSet ETH1_L1_ETH_GATEWAY_ADDRESS Proxy__OVM_L1ETHGateway
    if [ $ETH1_L1_ETH_GATEWAY_ADDRESS == null ]; then
        envSet ETH1_L1_ETH_GATEWAY_ADDRESS OVM_L1ETHGateway
    fi
fi

# wait for the dtl to be up, else geth will crash if it cannot connect
curl \
    --fail \
    --show-error \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    $ROLLUP_CLIENT_HTTP

exec geth --verbosity="$VERBOSITY" "$@"
