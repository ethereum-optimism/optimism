#!/bin/sh

# Get relative directory of this script
SCRIPT_DIR="$( dirname -- ${BASH_SOURCE[0]} )"

# -- RPCs --
L1_RPC="https://goerli-l1-8104952.optimism.io"
L2_RPC="https://goerli-3319642-sequencer.optimism.io"

# -- Message File Paths --
OVM_MESSAGES="$SCRIPT_DIR/data/messages/ovm-messages.json"
EVM_MESSAGES="$SCRIPT_DIR/data/messages/evm-messages.json"

# -- Contracts --
PORTAL="0x7db2f4b1f880257a99e024647cead4e3ad63b665"
L1XDM="0x5086d1eef304eb5284a0f6720f79403b4e9be294"
L1BRIDGE="0x636Af16bf2f682dD3109e60102b8E1A089FedAa8"

# -- Genesis Block --
L2GENESIS=3319643

# -- Account --
# Pub Key: 0x4F3278d9FF0426E4b60653bee23D0a768E700672
SECRET="$(cat $SCRIPT_DIR/.secret)"

# Extract messages tar
tar -xvf $SCRIPT_DIR/data/messages.tgz -C $SCRIPT_DIR/data

# Run built withdrawals binary
go run $SCRIPT_DIR/main.go                     \
  --l1-rpc-url $L1_RPC                         \
  --l2-rpc-url $L2_RPC                         \
  --ovm-messages $OVM_MESSAGES                 \
  --evm-messages $EVM_MESSAGES                 \
  --optimism-portal-address $PORTAL            \
  --l1-crossdomain-messenger-address $L1XDM    \
  --l1-standard-bridge-address $L1BRIDGE       \
  --bedrock-transition-block-number $L2GENESIS \
  --private-key $SECRET

# Delete message JSON files
rm -rf $SCRIPT_DIR/data/messages
