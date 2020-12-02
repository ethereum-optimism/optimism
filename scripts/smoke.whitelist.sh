#!/bin/sh

function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

function test_whitelist {
    banner
    echo "*** SHOULD SUCCEED ***" 
    echo "TRANSFER FUNDS FROM $ACCOUNT0 TO $WHITELISTED" 
    echo "vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0/debit to=$WHITELISTED amount=$FUNDING_AMOUNT"
    vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0/debit to=$WHITELISTED amount=$FUNDING_AMOUNT
    banner
    vault write  -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0/debit to=$WHITELISTED amount=$FUNDING_AMOUNT

    banner
    echo "*** SHOULD FAIL! ***" 
    echo "TRANSFER FUNDS FROM $ACCOUNT0 TO $UNLISTED" 
    echo "vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0/debit to=$UNLISTED amount=$FUNDING_AMOUNT"
    vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0/debit to=$UNLISTED amount=$FUNDING_AMOUNT
    banner
    vault write  -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0/debit to=$UNLISTED amount=$FUNDING_AMOUNT

}
source /vault/scripts/smoke.env.sh

EMPTY=""
FUNDING_AMOUNT=100000000000000000
TEST_AMOUNT=10000000000000000

banner
echo "CONFIGURE MOUNT WITH NO WHITELIST"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"
banner
vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"

banner
echo "CREATE WALLET WITH MNEMONIC"
echo "vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet mnemonic='$MNEMONIC'"
vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet mnemonic="$MNEMONIC"
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet mnemonic="$MNEMONIC"

banner
echo "CREATE NEW ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/whitelist-wallet/accounts"
ACCOUNT0=$(vault write -f -field=address immutability-eth-plugin/wallets/whitelist-wallet/accounts)
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet/accounts

banner
echo "CREATE NEW WHITELISTED ACCOUNT"
echo "vault write -format=json -f immutability-eth-plugin/wallets/whitelist-wallet/accounts"
WHITELISTED=$(vault write -f -field=address immutability-eth-plugin/wallets/whitelist-wallet/accounts)
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet/accounts

banner
echo "CREATE NEW UNLISTED ACCOUNT"
echo "vault write -format=json -f immutability-eth-plugin/wallets/whitelist-wallet/accounts"
UNLISTED=$(vault write -f -field=address immutability-eth-plugin/wallets/whitelist-wallet/accounts)
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet/accounts

banner
echo "ACCOUNT-LEVEL WHITELIST: ADD $WHITELISTED TO WHITELIST FOR $ACCOUNT0"
echo "vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0 whitelist=$WHITELISTED"
vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0 whitelist=$WHITELISTED
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0 whitelist=$WHITELISTED

test_whitelist

banner
echo "ACCOUNT-LEVEL WHITELIST: REMOVE WHITELIST FOR $ACCOUNT0"
echo "vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0 whitelist=$EMPTY"
vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0 whitelist=$EMPTY
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet/accounts/$ACCOUNT0 whitelist=$EMPTY

banner
echo "WALLET-LEVEL WHITELIST: ADD $WHITELISTED TO WHITELIST FOR whitelist-wallet"
echo "vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet whitelist=$WHITELISTED"
vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet whitelist=$WHITELISTED
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet whitelist=$WHITELISTED

test_whitelist

banner
echo "WALLET-LEVEL WHITELIST: REMOVE WHITELIST FOR whitelist-wallet"
echo "vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet whitelist=$EMPTY"
vault write -format=json immutability-eth-plugin/wallets/whitelist-wallet whitelist=$EMPTY
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/whitelist-wallet whitelist=$EMPTY


banner
echo "GLOBAL-LEVEL WHITELIST: ADD $WHITELISTED TO WHITELIST FOR immutability-eth-plugin"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID' whitelist='$WHITELISTED'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" whitelist=$WHITELISTED
banner
vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" whitelist=$WHITELISTED

test_whitelist

banner
echo "GLOBAL-LEVEL WHITELIST: REMOVE WHITELIST FOR immutability-eth-plugin"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID' whitelist='$EMPTY'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" whitelist=$EMPTY
banner
vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" whitelist=$EMPTY
