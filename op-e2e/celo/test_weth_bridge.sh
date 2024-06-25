#!/bin/bash
#shellcheck disable=SC2086
set -eo pipefail
set -x

source shared.sh
SCRIPT_DIR=$(readlink -f "$(dirname "$0")")
CONTRACTS_DIR=$SCRIPT_DIR/../../packages/contracts-bedrock

# Deploy WETH
L1_WETH=$(
  ETH_RPC_URL=$ETH_RPC_URL_L1 forge create --private-key=$ACC_PRIVKEY --root $CONTRACTS_DIR $CONTRACTS_DIR/src/dispute/weth/WETH98.sol:WETH98 --json | jq .deployedTo -r
)

# create ERC20 token on L2
L2_TOKEN=$(
  cast send --private-key $ACC_PRIVKEY 0x4200000000000000000000000000000000000012 "createOptimismMintableERC20(address,string,string)" $L1_WETH "Wrapped Ether" "WETH" --json \
    | jq -r '.logs[0].topics[2]' | cast parse-bytes32-address
)

# Wrap some ETH
ETH_RPC_URL=$ETH_RPC_URL_L1 cast send --private-key $ACC_PRIVKEY $L1_WETH --value 1ether
# Approve transfer to bridge
L1_BRIDGE_ADDR=$(cast call 0x4200000000000000000000000000000000000010 'otherBridge() returns (address)')
ETH_RPC_URL=$ETH_RPC_URL_L1 cast send --private-key $ACC_PRIVKEY $L1_WETH 'approve(address, uint256) returns (bool)' $L1_BRIDGE_ADDR 1ether
# Bridge to L2
ETH_RPC_URL=$ETH_RPC_URL_L1 cast send --private-key $ACC_PRIVKEY $L1_BRIDGE_ADDR 'bridgeERC20(address _localToken, address _remoteToken, uint256 _amount, uint32 _minGasLimit, bytes calldata _extraData)' $L1_WETH $L2_TOKEN 0.3ether 50000 0x --gas-limit 6000000

# Setup up oracle and FeeCurrencyDirectory
ORACLE=$(forge create --private-key=$ACC_PRIVKEY --root $CONTRACTS_DIR $CONTRACTS_DIR/src/celo/testing/MockSortedOracles.sol:MockSortedOracles --json | jq .deployedTo -r)
cast send --private-key $ACC_PRIVKEY $ORACLE 'setMedianRate(address, uint256)' $L2_TOKEN 100000000000000000
cast send --private-key $ACC_PRIVKEY $FEE_CURRENCY_DIRECTORY_ADDR 'setCurrencyConfig(address, address, uint256)' $L2_TOKEN $ORACLE 60000

# Check balance from bridging (we intentionally don't do this right after bridging, since it takes a bit)
L2_BALANCE=$(cast call $L2_TOKEN 'balanceOf(address) returns (uint256)' $ACC_ADDR)
echo L2 balance: $L2_BALANCE
[[ $(echo $L2_BALANCE | awk '{print $1}') -gt 0 ]] || (echo "Bridging to L2 failed!"; exit 1)

# Send fee currency tx!
#TXHASH=$(~/op-geth/e2e_test/js-tests/send_tx.mjs 901 $ACC_PRIVKEY $L2_TOKEN)
#cast receipt $TXHASH
echo You can use privkey $ACC_PRIVKEY to pay for txs with $L2_TOKEN, now.
