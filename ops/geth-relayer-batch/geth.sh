#!/bin/bash
#set -e
RETRIES=${RETRIES:-40}
VERBOSITY=${VERBOSITY:-6}

function envSet() {
    VAR=$1
    export $VAR=$(echo $ADDRESSES | jq -r ".$2")
}

echo "URL => $URL">> /app/log/t_geth.log
if [[ ! -z "$URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
    
    # set all the necessary env vars
    envSet ETH1_ADDRESS_RESOLVER_ADDRESS AddressManager
    envSet ETH1_L1_CROSS_DOMAIN_MESSENGER_ADDRESS Proxy__OVM_L1CrossDomainMessenger
    envSet ROLLUP_ADDRESS_MANAGER_OWNER_ADDRESS Deployer

    # set the address to the proxy gateway if possible
    envSet ETH1_L1_ETH_GATEWAY_ADDRESS Proxy__OVM_L1ETHGateway
    if [ $ETH1_L1_ETH_GATEWAY_ADDRESS == null ]; then
        envSet ETH1_L1_ETH_GATEWAY_ADDRESS OVM_L1ETHGateway
    fi
    envSet MVM_L1GATEWAY_ADDRESS Proxy__MVM_L1MetisGateway
    if [ $ETH1_L1_ETH_GATEWAY_ADDRESS == null ]; then
        envSet MVM_L1GATEWAY_ADDRESS MVM_L1MetisGateway
    fi
fi

# wait for the dtl to be up, else geth will crash if it cannot connect
echo "ROLLUP_CLIENT_HTTP => $ROLLUP_CLIENT_HTTP">> /app/log/t_geth.log
CMD="$ROLLUP_CLIENT_HTTP/eth/syncing/$CHAIN_ID"
echo "CMD => $CMD">> /app/log/t_geth.log
curl \
    --fail \
    --show-error \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    $CMD

#exec geth --verbosity="$VERBOSITY" "$@"
nohup geth --verbosity="$VERBOSITY" "$@" >> /app/log/t_geth.log &
