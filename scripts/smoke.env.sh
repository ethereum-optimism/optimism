#!/bin/sh

export MNEMONIC="explain foam nice clown method avocado hill basket echo blur elevator marble"
export CHAIN_ID=5777
export CHAIN_L2_ID=28
export PORT=8545
export RPC_URL="http://ganache:$PORT"
export RPC_L2_URL="http://l2geth:$PORT"
export CONTRACTS_PATH="/vault/contracts/erc20/build/"
export PLASMA_CONTRACT=`cat /truffleshuffle/plasma_framework_addr.out`
export GAS_PRICE_LOW="1"
export GAS_PRICE_HIGH="37000000000"
export FUNDING_AMOUNT=100000000000000000
export TEST_AMOUNT=10000000000000000
export PASSPHRASE="passion bauble hypnotic hanky kiwi effective overcast roman staleness"
export EMPTY=""
export BLOCK_ROOT="KW7c+YhqaeXzUSARcnOh0sBSWhAU7l144fF6ls0Y5Vw="
export BAD_BLOCK_ROOT="KW7c+YhqaeXzUSARcnOh0sBSWhAU7l144fF6ls0Y"
function check_result(){
  EXIT_STATUS=$1
  EXPECTED=$2
  echo "Exit status of command was $EXIT_STATUS."
  [[ $EXIT_STATUS -ne $EXPECTED ]] && echo 'DID NOT PASS THE REQUIRED TEST' && exit $EXIT_STATUS
}
function check_string_result(){
  EXIT_STRING=$1
  EXPECTED=$2
  echo "Exit string of command was $EXIT_STRING."
  echo "Expected output of command was $EXPECTED."
  [[ $EXIT_STRING != $EXPECTED ]] && echo 'DID NOT PASS THE REQUIRED TEST' && exit 1
}