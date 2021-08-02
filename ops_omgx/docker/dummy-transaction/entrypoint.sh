#!/bin/sh
RETRIES=${RETRIES:-40}
export NODE_ENV=`/opt/secret2env -name $SECRETNAME|grep -w NODE_ENV|sed 's/NODE_ENV=//g'`
export L1_NODE_WEB3_WS=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_WS|sed 's/L1_NODE_WEB3_WS=//g'`
export L1_LIQUIDITY_POOL_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w L1_LIQUIDITY_POOL_ADDRESS|sed 's/L1_LIQUIDITY_POOL_ADDRESS=//g'`
export L2_LIQUIDITY_POOL_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w L2_LIQUIDITY_POOL_ADDRESS|sed 's/L2_LIQUIDITY_POOL_ADDRESS=//g'`
export RELAYER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w RELAYER_ADDRESS|sed 's/RELAYER_ADDRESS=//g'`
export SEQUENCER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w SEQUENCER_ADDRESS|sed 's/SEQUENCER_ADDRESS=//g'`
export L2_DEPOSITED_ERC20=0x0e52DEfc53ec6dCc52d630af949a9b6313455aDF
export DUMMY_DELAY_MINS=5
export DUMMY_ETH_AMOUNT=0.0005
export DUMMY_TIMEOUT_MINS=1
if [[ ! -z "$URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
    # set the env
    export L1_ADDRESS_MANAGER=$(echo $ADDRESSES | jq -r '.AddressManager')
fi
npm run dummy-transaction
