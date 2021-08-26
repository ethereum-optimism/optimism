#!/bin/bash
export MYSQL_HOST_URL=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_HOST_URL|sed 's/MYSQL_HOST_URL=//g'`
export MYSQL_PORT=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_PORT|sed 's/MYSQL_PORT=//g'`
export MYSQL_USERNAME=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_USERNAME|sed 's/MYSQL_USERNAME=//g'`
export MYSQL_PASSWORD=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_PASSWORD|sed 's/MYSQL_PASSWORD=//g'`
export MYSQL_DATABASE_NAME=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_DATABASE_NAME|sed 's/MYSQL_DATABASE_NAME=//g'`
export ADDRESS_MANAGER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w ADDRESS_MANAGER_ADDRESS|sed 's/ADDRESS_MANAGER_ADDRESS=//g'`
export L2_MESSENGER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w L2_MESSENGER_ADDRESS|sed 's/L2_MESSENGER_ADDRESS=//g'`
export DEPLOYER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w DEPLOYER_PRIVATE_KEY|sed 's/DEPLOYER_PRIVATE_KEY=//g'`
export TRANSACTION_MONITOR_INTERVAL=`/opt/secret2env -name $SECRETNAME|grep -w TRANSACTION_MONITOR_INTERVAL|sed 's/TRANSACTION_MONITOR_INTERVAL=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`

/usr/local/bin/run-monitor.js
