#!/bin/sh

function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

source /vault/scripts/smoke.env.sh

banner
echo "CONFIGURE MOUNT"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID' rpc_l2_url='$RPC_L2_URL' chain_l2_id='$CHAIN_L2_ID'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"
check_result $? 0
banner
vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"

banner
echo "READ MOUNT CONFIGURATION"
echo "vault read -format=json immutability-eth-plugin/config"
vault read -format=json immutability-eth-plugin/config
check_result $? 0
banner
vault read  -output-curl-string immutability-eth-plugin/config

banner
echo "CREATE WALLET WITHOUT MNEMONIC"
echo "vault write -format=json -f immutability-eth-plugin/wallets/test-wallet-1"
vault write -format=json -f immutability-eth-plugin/wallets/test-wallet-1
check_result $? 0
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-1

banner
echo "CREATE TEMPORARY WALLET WITHOUT MNEMONIC"
echo "vault write -format=json -f immutability-eth-plugin/wallets/temp-wallet"
vault write -format=json -f immutability-eth-plugin/wallets/temp-wallet
check_result $? 0
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/temp-wallet

banner
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
check_result $? 0
banner
vault list -output-curl-string immutability-eth-plugin/wallets

banner
echo "CREATE WALLET WITH MNEMONIC"
echo "vault write -format=json immutability-eth-plugin/wallets/test-wallet-2 mnemonic='$MNEMONIC'"
vault write -format=json immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"
check_result $? 0
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"

banner
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
check_result $? 0
banner
vault list -output-curl-string immutability-eth-plugin/wallets

banner
echo "CREATE NEW ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT0=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts

banner
echo "CREATE SECOND NEW ACCOUNT IN WALLET"
echo "vault write -format=json immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT1=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
banner
vault write  -output-curl-string -f immutability-eth-plugin/wallets/test-wallet-2/accounts

banner
echo "TRANSFER FUNDS FROM $ACCOUNT0 TO $ACCOUNT1" 
echo "vault write -format=json immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT"
vault write -format=json immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT
check_result $? 0
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT

banner
echo "CREATE TEMPORARY NEW ACCOUNT IN WALLET"
echo "vault write -format=json immutability-eth-plugin/wallets/test-wallet-2/accounts"
TEMP_ACCOUNT=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
check_result $? 0
banner
vault write  -output-curl-string -f immutability-eth-plugin/wallets/test-wallet-2/accounts
