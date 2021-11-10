#!/bin/bash

# Deterministically recreate the gas price oracle bindings
# for testing. This script depends on geth being in the monorepo

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
ABIGEN="$SCRIPTS_DIR/../cmd/abigen/main.go"
CONTRACTS_PATH="$SCRIPTS_DIR/../../packages/contracts/artifacts/contracts"
GAS_PRICE_ORACLE="$CONTRACTS_PATH/L2/predeploys/OVM_GasPriceOracle.sol/OVM_GasPriceOracle.json"

OUT_DIR="$SCRIPTS_DIR/../rollup/fees/bindings"
mkdir -p $OUT_DIR

tmp=$(mktemp)

cat $GAS_PRICE_ORACLE | jq -r .bytecode > $tmp

cat $GAS_PRICE_ORACLE \
    | jq .abi \
    | go run $ABIGEN --pkg bindings \
    --abi - \
    --out $OUT_DIR/gaspriceoracle.go \
    --type GasPriceOracle \
    --bin "$tmp"

rm $tmp
