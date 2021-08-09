#!/bin/sh

# FIXME: Cannot use set -e since bash is not installed in Dockerfile
# set -e

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
    
    envSet MVM_L1GATEWAY_ADDRESS Proxy__OVM_L1ETHGateway
    if [ $ETH1_L1_ETH_GATEWAY_ADDRESS == null ]; then
        envSet ETH1_L1_ETH_GATEWAY_ADDRESS OVM_L1ETHGateway
    fi
fi

JSON='{"jsonrpc":"2.0","id":0,"method":"admin_nodeInfo","params":[]}'
NODE_INFO=$(curl --silent --fail --show-error -H "Content-Type: application/json" --retry-connrefused --retry $RETRIES --retry-delay 3  -d $JSON $L2_URL)

NODE_ENODE=$(echo $NODE_INFO | jq -r '.result.enode')
NODE_IP=$(echo $NODE_INFO | jq -r '.result.ip')

if [ "$NODE_IP" = "127.0.0.1" ];then
    HOST_IP=$(/sbin/ip route | awk '/default/ { print $3 }')
    NODE_ENODE=${NODE_ENODE//127.0.0.1/$HOST_IP}
fi

mkdir $(echo $DATADIR)
touch $(echo $DATADIR)/static-nodes.json

echo "[\"$NODE_ENODE\"]" > $(echo $DATADIR)/static-nodes.json

exec geth --verbosity="$VERBOSITY" "$@"
