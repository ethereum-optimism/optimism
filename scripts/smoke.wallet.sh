#!/bin/sh

function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

source /home/vault/scripts/smoke.env.sh

PASSPHRASE="passion bauble hypnotic hanky kiwi effective overcast roman staleness"
FUNDING_AMOUNT=100000000000000000
TEST_AMOUNT=10000000000000000

banner
echo "CONFIGURE MOUNT"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"
banner
vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"

banner
echo "READ MOUNT CONFIGURATION"
echo "vault read -format=json immutability-eth-plugin/config"
vault read -format=json immutability-eth-plugin/config
banner
vault read  -output-curl-string immutability-eth-plugin/config

banner
echo "CREATE WALLET WITHOUT MNEMONIC"
echo "vault write -format=json -f immutability-eth-plugin/wallets/test-wallet-1"
vault write -format=json -f immutability-eth-plugin/wallets/test-wallet-1
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-1

banner
echo "CREATE TEMPORARY WALLET WITHOUT MNEMONIC"
echo "vault write -format=json -f immutability-eth-plugin/wallets/temp-wallet"
vault write -format=json -f immutability-eth-plugin/wallets/temp-wallet
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/temp-wallet

banner
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
banner
vault list -output-curl-string immutability-eth-plugin/wallets

banner
echo "CREATE WALLET WITH MNEMONIC"
echo "vault write -format=json immutability-eth-plugin/wallets/test-wallet-2 mnemonic='$MNEMONIC'"
vault write -format=json immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"

banner
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
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
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT

banner
echo "CREATE TEMPORARY NEW ACCOUNT IN WALLET"
echo "vault write -format=json immutability-eth-plugin/wallets/test-wallet-2/accounts"
TEMP_ACCOUNT=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
banner
vault write  -output-curl-string -f immutability-eth-plugin/wallets/test-wallet-2/accounts