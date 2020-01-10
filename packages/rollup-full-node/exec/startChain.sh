#!/bin/bash

# Assumes you have parity installed. If not, install it by running the following:
# brew tap paritytech/paritytech
# brew install parity

BASE_DIR=$(dirname $0)
LOG_DIR=$BASE_DIR/../log
mkdir -p $LOG_DIR

parity --chain $BASE_DIR/../config/parity/local-chain-config.json --min-gas-price 0  2>&1 | tee $LOG_DIR/parity.$(date '+%Y.%m.%d_%H.%M.%S')_$(uuidgen).log
