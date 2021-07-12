#!/bin/sh

source /vault/scripts/smoke.env.sh


echo ""
echo "------------------------------------------------------------------"
echo "CONFIGURE MOUNT"
echo "vault write -f immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='5777' rpc_l2_url='$RPC_L2_URL' chain_l2_id='$CHAIN_L2_ID'"
vault write immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"
check_result $? 0
#vault write -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"

echo ""
echo "------------------------------------------------------------------"
echo "READ MOUNT CONFIGURATION"
echo "vault read immutability-eth-plugin/config"
vault read immutability-eth-plugin/config
check_result $? 0
#vault read -output-curl-string immutability-eth-plugin/config

echo ""
echo "------------------------------------------------------------------"
echo "CREATE WALLET WITHOUT MNEMONIC"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-1"
vault write -f immutability-eth-plugin/wallets/test-wallet-1
check_result $? 0
#vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-1

echo ""
echo "------------------------------------------------------------------"
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
check_result $? 0
#vault list -output-curl-string immutability-eth-plugin/wallets

echo ""
echo "------------------------------------------------------------------"
echo "CREATE WALLET WITH MNEMONIC"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2 mnemonic='$MNEMONIC'"
vault write immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"
check_result $? 0
#vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"

echo ""
echo "------------------------------------------------------------------"
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
check_result $? 0
#vault list -output-curl-string immutability-eth-plugin/wallets

echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN WALLET"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT0=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
check_result $? 0
#vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts

echo ""
echo "------------------------------------------------------------------"
echo "CREATE SECOND NEW ACCOUNT IN WALLET WITH WHITELIST == $ACCOUNT0"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts whitelist=$ACCOUNT0"
ACCOUNT1=$(vault write -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts whitelist=$ACCOUNT0)
check_result $? 0
#vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts whitelist=$ACCOUNT0

echo ""
echo "------------------------------------------------------------------"
echo "CREATE THIRD NEW ACCOUNT IN WALLET WITH BLACKLIST == $ACCOUNT1"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts blacklist=$ACCOUNT1"
ACCOUNT2=$(vault write -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts blacklist=$ACCOUNT1)
check_result $? 0
#vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts blacklist=$ACCOUNT1

echo ""
echo "------------------------------------------------------------------"
echo "FUND ACCOUNTS ($ACCOUNT1, $ACCOUNT2) FROM $ACCOUNT0" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT
check_result $? 0
#vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT

echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT2 amount=$FUNDING_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT2 amount=$FUNDING_AMOUNT
check_result $? 0
echo ""
echo "------------------------------------------------------------------"
echo "VIOLATE $ACCOUNT1'S WHITELIST BY ATTEMPTING TO SEND TO $ACCOUNT2" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT2 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT2 amount=$TEST_AMOUNT
check_result $? 2
#vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT2 amount=$TEST_AMOUNT

echo ""
echo "------------------------------------------------------------------"
echo "ADHERE TO $ACCOUNT1'S WHITELIST BY ATTEMPTING TO SEND TO $ACCOUNT0" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT0 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT0 amount=$TEST_AMOUNT
check_result $? 0
# vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT0 amount=$TEST_AMOUNT

echo ""
echo "------------------------------------------------------------------"
echo "VIOLATE $ACCOUNT2'S BLACKLIST BY ATTEMPTING TO SEND TO $ACCOUNT1" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT1 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT1 amount=$TEST_AMOUNT
check_result $? 2

# vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT1 amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "ADHERE TO $ACCOUNT2'S BLACKLIST BY ATTEMPTING TO SEND TO $ACCOUNT0" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT0 amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT0 amount=$TEST_AMOUNT
check_result $? 0

# vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT0 amount=$TEST_AMOUNT
echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN WALLET TO TEST WHITELIST AT WALLET LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_WALLET_WHITE=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
check_result $? 0

echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN WALLET TO TEST BLACKLISTS AT WALLET LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_WALLET_BLACK=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
check_result $? 0

echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN test-wallet-2 TO TEST WHITELISTS AND BLACKLISTS AT WALLET LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_NOT_WHITE=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
check_result $? 0

echo ""
echo "------------------------------------------------------------------"
echo "CREATE NEW ACCOUNT IN test-wallet-2 TO TEST WHITELISTS AND BLACKLISTS AT GLOBAL LEVEL"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT_BIG_BAD=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
check_result $? 0

echo ""
echo "------------------------------------------------------------------"
echo "test-wallet-2 HAS NO BLACKLIST YET SO SEND TO $ACCOUNT_WALLET_BLACK" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT
check_result $? 0

###
### // whitelisting and blacklisting are not implemented in this release!
### which is what is tested below!

# echo ""
# echo "------------------------------------------------------------------"
# echo "SET test-wallet-2 BLACKLIST"
# echo "vault write immutability-eth-plugin/wallets/test-wallet-2 blacklist=$ACCOUNT_WALLET_BLACK"
# vault write immutability-eth-plugin/wallets/test-wallet-2 blacklist=$ACCOUNT_WALLET_BLACK
# check_result $? 0
# # vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 blacklist=$ACCOUNT_WALLET_BLACK

# echo ""
# echo "------------------------------------------------------------------"
# echo "VIOLATE TO test-wallet-2's BLACKLIST BY ATTEMPTING TO SEND TO $ACCOUNT_WALLET_BLACK" 
# echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT"
# vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT
# check_result $? 0
# # vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT2/debit to=$ACCOUNT_WALLET_BLACK amount=$TEST_AMOUNT

# echo ""
# echo "------------------------------------------------------------------"
# echo "ATTEMPT TO SEND TO $ACCOUNT_WALLET_WHITE - SHOULD FAIL - NOT WHITELISTED BY $ACCOUNT1" 
# echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT"
# vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT
# check_result $? 0
# # vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT

# echo ""
# echo "------------------------------------------------------------------"
# echo "SET test-wallet-2 WHITELIST AND BLACKLIST"
# echo "vault write immutability-eth-plugin/wallets/test-wallet-2 whitelist=$ACCOUNT_WALLET_WHITE blacklist=$ACCOUNT_WALLET_BLACK"
# vault write immutability-eth-plugin/wallets/test-wallet-2 whitelist=$ACCOUNT_WALLET_WHITE blacklist=$ACCOUNT_WALLET_BLACK
# check_result $? 0
# # vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 whitelist=$ACCOUNT_WALLET_WHITE blacklist=$ACCOUNT_WALLET_BLACK

# echo ""
# echo "------------------------------------------------------------------"
# echo "ATTEMPT TO SEND TO $ACCOUNT_WALLET_WHITE - SHOULD SUCCEED - WHITELISTED BY WALLET" 
# echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT"
# vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT
# check_result $? 0
# #vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_WALLET_WHITE amount=$TEST_AMOUNT

###
### and these (below) work
###

echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_NOT_WHITE - SHOULD FAIL - NOT WHITELISTED BY WALLET OR ACCOUNT" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT
check_result $? 2
# vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT

echo ""
echo "------------------------------------------------------------------"
echo "ADD $ACCOUNT_NOT_WHITE TO THE GLOBAL WHITELIST"
echo "vault write immutability-eth-plugin/config whitelist=$ACCOUNT_NOT_WHITE rpc_url=$RPC_URL chain_id=$CHAIN_ID rpc_l2_url=$RPC_L2_URL chain_l2_id=$CHAIN_L2_ID"
vault write immutability-eth-plugin/config whitelist=$ACCOUNT_NOT_WHITE rpc_url=$RPC_URL chain_id=$CHAIN_ID rpc_l2_url=$RPC_L2_URL chain_l2_id=$CHAIN_L2_ID
check_result $? 0
# vault write -output-curl-string immutability-eth-plugin/config whitelist=$ACCOUNT_NOT_WHITE rpc_url=$RPC_URL chain_id=$CHAIN_ID rpc_l2_url=$RPC_L2_URL chain_l2_id=$CHAIN_L2_ID

echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_NOT_WHITE - SHOULD SUCCEED - GLOBALLY WHITELISTED" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT
check_result $? 0
# vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_NOT_WHITE amount=$TEST_AMOUNT

echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_BIG_BAD - SHOULD FAIL - NOT WHITELISTED BY WALLET OR ACCOUNT" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT
check_result $? 2

echo ""
echo "------------------------------------------------------------------"
echo "ADD $ACCOUNT_BIG_BAD TO THE GLOBAL WHITELIST"
echo "vault write immutability-eth-plugin/config whitelist=$ACCOUNT_BIG_BAD rpc_url=$RPC_URL chain_id=$CHAIN_ID rpc_l2_url=$RPC_L2_URL chain_l2_id=$CHAIN_L2_ID"
vault write immutability-eth-plugin/config whitelist=$ACCOUNT_BIG_BAD rpc_url=$RPC_URL chain_id=$CHAIN_ID rpc_l2_url=$RPC_L2_URL chain_l2_id=$CHAIN_L2_ID
check_result $? 0

echo ""
echo "------------------------------------------------------------------"
echo "ATTEMPT TO SEND TO $ACCOUNT_BIG_BAD - SHOULD SUCCEED - GLOBALLY WHITELISTED" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/debit to=$ACCOUNT_BIG_BAD amount=$TEST_AMOUNT
check_result $? 0
echo ""
