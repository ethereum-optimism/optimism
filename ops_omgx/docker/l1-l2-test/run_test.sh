#!/bin/bash

export ETH1_ADDRESS_RESOLVER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w ADDRESS_MANAGER_ADDRESS|sed 's/ADDRESS_MANAGER_ADDRESS=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`
export TEST_PRIVATE_KEY_1=`/opt/secret2env -name $SECRETNAME|grep -w TEST_PRIVATE_KEY_1|sed 's/TEST_PRIVATE_KEY_1=//g'`
export TARGET_GAS_LIMIT=9000000000
export CHAIN_ID=`/opt/secret2env -name $SECRETNAME|grep -w CHAIN_ID|sed 's/CHAIN_ID=//g'`
export GASPRICE=`/opt/secret2env -name $SECRETNAME|grep -w 15000000|sed 's/15000000=//g'`

cd /opt/optimism/ops_omgx/test/
yarn test
