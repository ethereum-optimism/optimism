#!/bin/sh

MNEMONIC="explain foam nice clown method avocado hill basket echo blur elevator marble"

CHAIN_ID=5777
PORT=8545
RPC_URL="http://ganache:$PORT"
PASSPHRASE="passion bauble hypnotic hanky kiwi effective overcast roman staleness"
FUNDING_AMOUNT=100000000000000000
TEST_AMOUNT=10000000000000000

CONTRACTS_PATH="/home/vault/contracts/erc20/build/"
CONTRACT_SAFE_MATH="SafeMath"
CONTRACT_OWNED="Owned"
CONTRACT_FIXED_SUPPLY_TOKEN="FixedSupplyToken"
BIN_FILE=".bin"
ABI_FILE=".abi"

echo ""
echo "------------------------------------------------------------------"
echo "CONFIGURE MOUNT"
echo "vault write -f immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='5777'"
vault write immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"

vault write -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"
echo ""
echo "------------------------------------------------------------------"
echo "CREATE WALLET WITHOUT MNEMONIC"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-1"
vault write -f immutability-eth-plugin/wallets/test-wallet-1

vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-1
echo ""
echo "------------------------------------------------------------------"
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets

vault list -output-curl-string immutability-eth-plugin/wallets
echo ""
echo "------------------------------------------------------------------"
echo "CREATE WALLET WITH MNEMONIC"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2 mnemonic='$MNEMONIC'"
vault write immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"
echo ""
echo "------------------------------------------------------------------"
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets

vault list -output-curl-string immutability-eth-plugin/wallets
echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN WALLET"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT0=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)

vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts
echo ""
echo "------------------------------------------------------------------"
echo "CHECK BALANCE FOR ACCOUNT IN WALLET"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance

vault read -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance
echo ""
echo "------------------------------------------------------------------"
echo "CHECK BALANCE FOR ACCOUNT IN WALLET"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance
echo ""
echo "------------------------------------------------------------------"
echo "CHECK BALANCE FOR ACCOUNT IN WALLET"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance
echo ""
echo "------------------------------------------------------------------"
echo "CREATE SECOND NEW ACCOUNT IN WALLET WITH WHITELIST == $ACCOUNT0"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts whitelist=$ACCOUNT0"
ACCOUNT1=$(vault write -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts whitelist=$ACCOUNT0)

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts whitelist=$ACCOUNT0
echo ""
echo "------------------------------------------------------------------"
echo "CREATE THIRD NEW ACCOUNT IN WALLET WITH BLACKLIST == $ACCOUNT1"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts blacklist=$ACCOUNT1"
ACCOUNT2=$(vault write -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts blacklist=$ACCOUNT1)

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts blacklist=$ACCOUNT1
echo ""
echo "------------------------------------------------------------------"
echo "FUND ACCOUNTS ($ACCOUNT1, $ACCOUNT2) FROM $ACCOUNT0" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT2 amount=$FUNDING_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT2 amount=$FUNDING_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "VIOLATE $ACCOUNT1'S WHITELIST BY ATTEMPTING TO SEND TO $ACCOUNT2" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT2 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT2 amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT2 amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ADHERE TO $ACCOUNT1'S WHITELIST BY ATTEMPTING TO SEND TO $ACCOUNT0" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT0 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT0 amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT0 amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "VIOLATE $ACCOUNT2'S BLACKLIST BY ATTEMPTING TO SEND TO $ACCOUNT1" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT1 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT1 amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT1 amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ADHERE TO $ACCOUNT2'S BLACKLIST BY ATTEMPTING TO SEND TO $ACCOUNT0" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT0 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT0 amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT0 amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN WALLET TO TEST WHITELIST AT WALLET LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_WALLET_WHITE=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)

vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts
echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN WALLET TO TEST BLACKLISTS AT WALLET LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_WALLET_BLACK=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)

vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts
echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN test-wallet-2 TO TEST WHITELISTS AND BLACKLISTS AT WALLET LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_NOT_WHITE=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)

vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts

echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN test-wallet-2 TO TEST WHITELISTS AND BLACKLISTS AT GLOBAL LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_BIG_BAD=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)

vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts

echo ""
echo "------------------------------------------------------------------"
echo "test-wallet-2 HAS NO BLACKLIST YET SO SEND TO $ACCOUNT_WALLET_BLACK" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "SET test-wallet-2 BLACKLIST"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2 blacklist=$ACCOUNT_WALLET_BLACK"
vault write immutability-eth-plugin/wallets/test-wallet-2 blacklist=$ACCOUNT_WALLET_BLACK

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 blacklist=$ACCOUNT_WALLET_BLACK
echo ""
echo "------------------------------------------------------------------"
echo "VIOLATE TO test-wallet-2's BLACKLIST BY ATTEMPTING TO SEND TO $ACCOUNT_WALLET_BLACK" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_WALLET_WHITE - SHOULD FAIL - NOT WHITELISTED BY $ACCOUNT1" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "SET test-wallet-2 WHITELIST AND BLACKLIST"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2 whitelist=$ACCOUNT_WALLET_WHITE blacklist=$ACCOUNT_WALLET_BLACK"
vault write immutability-eth-plugin/wallets/test-wallet-2 whitelist=$ACCOUNT_WALLET_WHITE blacklist=$ACCOUNT_WALLET_BLACK

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 whitelist=$ACCOUNT_WALLET_WHITE blacklist=$ACCOUNT_WALLET_BLACK
echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_WALLET_WHITE - SHOULD SUCCEED - WHITELISTED BY WALLET" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_NOT_WHITE - SHOULD FAIL - NOT WHITELISTED BY WALLET OR ACCOUNT" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ADD $ACCOUNT_NOT_WHITE TO THE GLOBAL WHITELIST"
echo "vault write immutability-eth-plugin/config whitelist=$ACCOUNT_NOT_WHITE rpc_url=$RPC_URL chain_id=$CHAIN_ID"
vault write immutability-eth-plugin/config whitelist=$ACCOUNT_NOT_WHITE rpc_url=$RPC_URL chain_id=$CHAIN_ID

vault write -output-curl-string immutability-eth-plugin/config whitelist=$ACCOUNT_NOT_WHITE rpc_url=$RPC_URL chain_id=$CHAIN_ID
echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_NOT_WHITE - SHOULD SUCCEED - GLOBALLY WHITELISTED" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT

vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_BIG_BAD - SHOULD FAIL - NOT WHITELISTED BY WALLET OR ACCOUNT" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ADD $ACCOUNT_BIG_BAD TO THE GLOBAL WHITELIST"
echo "vault write immutability-eth-plugin/config whitelist=$ACCOUNT_BIG_BAD rpc_url=$RPC_URL chain_id=$CHAIN_ID"
vault write immutability-eth-plugin/config whitelist=$ACCOUNT_BIG_BAD rpc_url=$RPC_URL chain_id=$CHAIN_ID
echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_BIG_BAD - SHOULD SUCCEED - GLOBALLY WHITELISTED" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT
echo ""

echo "------------------------------------------------------------------"
echo "DEPLOY CONTRACT $CONTRACT_FIXED_SUPPLY_TOKEN"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/deploy abi=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$BIN_FILE"
CONTRACT=$(vault write -field=contract immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/deploy abi=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$BIN_FILE)
echo ""
echo "------------------------------------------------------------------"
echo "DEPLOY CONTRACT $CONTRACT_OWNED"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/deploy abi=@$CONTRACTS_PATH$CONTRACT_OWNED$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_OWNED$BIN_FILE"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/deploy abi=@$CONTRACTS_PATH$CONTRACT_OWNED$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_OWNED$BIN_FILE
echo ""
echo "------------------------------------------------------------------"
echo "DEPLOY CONTRACT $CONTRACT_SAFE_MATH"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/deploy abi=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$BIN_FILE"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/deploy abi=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$BIN_FILE
echo ""
echo "------------------------------------------------------------------"
echo "SIGN RAW TX FROM ACCOUNT"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to='$ACCOUNT1' data='hello' amount=1000000000000000"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to="$ACCOUNT1" data="hello" amount=1000000000000000
echo ""
echo "------------------------------------------------------------------"
echo "SIGN RAW TX FROM ACCOUNT WITH HEX ENCODING OF DATA"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to='$ACCOUNT1' data='fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19' encoding='hex'  amount=1000000000000000"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to="$ACCOUNT1" data="fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19" encoding="hex"  amount=1000000000000000
echo ""
echo "------------------------------------------------------------------"
echo "READ TOKEN TOTAL SUPPLY"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/totalSupply contract='$CONTRACT'"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/totalSupply contract="$CONTRACT"
echo ""
echo "------------------------------------------------------------------"``
echo "READ TOKEN BALANCE AT ACCOUNT"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/balanceOf contract='$CONTRACT'"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/balanceOf contract="$CONTRACT"
echo ""
echo "------------------------------------------------------------------"
echo "TRANSFER TOKEN FROM ACCOUNT"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/transfer contract='$CONTRACT'  to='$ACCOUNT2' tokens=23"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/transfer contract="$CONTRACT" to="$ACCOUNT2" tokens=23
echo ""
echo "------------------------------------------------------------------"
echo "APPROVE TOKEN TRANSFER FROM ACCOUNT TO ANOTHER"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/approve contract='$CONTRACT' spender='$ACCOUNT2' tokens=230"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/erc-20/approve contract="$CONTRACT" spender="$ACCOUNT2" tokens=230
echo ""
