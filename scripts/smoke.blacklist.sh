#!/bin/sh

function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

function test_blacklist {
    banner
    echo "*** SHOULD FAIL! ***" 
    echo "TRANSFER FUNDS FROM $ACCOUNT0 TO $BLACKLISTED" 
    echo "vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0/debit to=$BLACKLISTED amount=$FUNDING_AMOUNT"
    vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0/debit to=$BLACKLISTED amount=$FUNDING_AMOUNT
    check_result $? 0
    banner
    #vault write  -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0/debit to=$BLACKLISTED amount=$FUNDING_AMOUNT

    banner
    echo "*** SHOULD SUCCEED ***" 
    echo "TRANSFER FUNDS FROM $ACCOUNT0 TO $UNLISTED" 
    echo "vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0/debit to=$UNLISTED amount=$FUNDING_AMOUNT"
    vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0/debit to=$UNLISTED amount=$FUNDING_AMOUNT
    check_result $? 0
    banner
    #vault write  -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0/debit to=$UNLISTED amount=$FUNDING_AMOUNT

}

source /vault/scripts/smoke.env.sh

banner
echo "CONFIGURE MOUNT WITH NO BLACKLIST"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID' rpc_l2_url='$RPC_L2_URL' chain_l2_id='$CHAIN_L2_ID'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"
check_result $? 0
banner
#vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"

banner
echo "CREATE WALLET WITH MNEMONIC"
echo "vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet mnemonic='$MNEMONIC'"
vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet mnemonic="$MNEMONIC"
check_result $? 0
banner
#vault write  -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet mnemonic="$MNEMONIC"

banner
echo "CREATE NEW ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/blacklist-wallet/accounts"
ACCOUNT0=$(vault write -f -field=address immutability-eth-plugin/wallets/blacklist-wallet/accounts)
check_result $? 0
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet/accounts

banner
echo "CREATE NEW BLACKLISTED ACCOUNT"
echo "vault write -format=json -f immutability-eth-plugin/wallets/blacklist-wallet/accounts"
BLACKLISTED=$(vault write -f -field=address immutability-eth-plugin/wallets/blacklist-wallet/accounts)
check_result $? 0
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet/accounts

banner
echo "CREATE NEW UNLISTED ACCOUNT"
echo "vault write -format=json -f immutability-eth-plugin/wallets/blacklist-wallet/accounts"
UNLISTED=$(vault write -f -field=address immutability-eth-plugin/wallets/blacklist-wallet/accounts)
check_result $? 0
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet/accounts

banner
echo "ACCOUNT-LEVEL BLACKLIST: ADD $BLACKLISTED TO BLACKLIST FOR $ACCOUNT0"
echo "vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0 blacklist=$BLACKLISTED"
vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0 blacklist=$BLACKLISTED
check_result $? 0
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0 blacklist=$BLACKLISTED

###
### // whitelisting and blacklisting are not implemented in this release!
### which is what is tested below!

# test_blacklist

# banner
# echo "ACCOUNT-LEVEL BLACKLIST: REMOVE BLACKLIST FOR $ACCOUNT0"
# echo "vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0 blacklist=$EMPTY"
# vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0 blacklist=$EMPTY
# check_result $? 0
# banner
# #vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet/accounts/$ACCOUNT0 blacklist=$EMPTY

# banner
# echo "WALLET-LEVEL BLACKLIST: ADD $BLACKLISTED TO BLACKLIST FOR blacklist-wallet"
# echo "vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet blacklist=$BLACKLISTED"
# vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet blacklist=$BLACKLISTED
# check_result $? 0
# banner
# #vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet blacklist=$BLACKLISTED

# test_blacklist

# banner
# echo "WALLET-LEVEL BLACKLIST: REMOVE BLACKLIST FOR blacklist-wallet"
# echo "vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet blacklist=$EMPTY"
# vault write -format=json immutability-eth-plugin/wallets/blacklist-wallet blacklist=$EMPTY
# check_result $? 0
# banner
# #vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/blacklist-wallet blacklist=$EMPTY


# banner
# echo "GLOBAL-LEVEL BLACKLIST: ADD $BLACKLISTED TO BLACKLIST FOR immutability-eth-plugin"
# echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID' blacklist='$BLACKLISTED' rpc_l2_url='$RPC_L2_URL' chain_l2_id='$CHAIN_L2_ID'"
# vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" blacklist=$BLACKLISTED rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"
# check_result $? 0
# banner
# #vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" blacklist=$BLACKLISTED

# test_blacklist

# banner
# echo "GLOBAL-LEVEL BLACKLIST: REMOVE BLACKLIST FOR immutability-eth-plugin"
# echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID' blacklist='$EMPTY' rpc_l2_url='$RPC_L2_URL' chain_l2_id='$CHAIN_L2_ID'"
# vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" blacklist=$EMPTY rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"
# check_result $? 0
# banner
# #vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" blacklist=$EMPTY
