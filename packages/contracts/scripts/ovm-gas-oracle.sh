#!/bin/sh

set -e

RPC_URL=${RPC_URL:-http://localhost:8545}
OVM_GAS_ORACLE=0x420000000000000000000000000000000000000F

function send_tx() {
    cast send --rpc-url $RPC_URL \
        --private-key $PRIVATE_KEY \
        --legacy \
        --gas-price 0 \
        $OVM_GAS_ORACLE \
        $1 \
        $2
}

function call() {
    cast call --rpc-url $RPC_URL \
        $OVM_GAS_ORACLE \
        $1
}

echo "Scalar:       $(call 'scalar()(uint256)')"
echo "L2 gas price: $(call 'gasPrice()(uint256)')"
echo "Overhead:     $(call 'overhead()(uint256)')"

if [[ ! -z $PRIVATE_KEY ]]; then
    if [[ ! -z $SCALAR ]]; then
        echo "Setting scalar to $SCALAR"
        send_tx 'setScalar(uint256)' $SCALAR
    fi

    if [[ ! -z $OVERHEAD ]]; then
        echo "Setting overhead to $OVERHEAD"
        send_tx 'setOverhead(uint256)' $OVERHEAD
    fi

    if [[ ! -z $L2_GAS_PRICE ]]; then
        echo "Setting L2 gas price to $L2_GAS_PRICE"
        send_tx 'setGasPrice(uint256)' $L2_GAS_PRICE
    fi
fi

