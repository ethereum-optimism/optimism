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
echo "vault write immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID'"
vault write immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"
banner
vault write -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"

banner
echo "CREATE WALLET WITHOUT MNEMONIC"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-1"
vault write -f immutability-eth-plugin/wallets/test-wallet-1
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-1

banner
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
banner
vault list -output-curl-string immutability-eth-plugin/wallets

banner
echo "CREATE WALLET WITH MNEMONIC"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2 mnemonic='$MNEMONIC'"
vault write immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"
banner
vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2 mnemonic="$MNEMONIC"

banner
echo "LIST WALLETS"
echo "vault list immutability-eth-plugin/wallets"
vault list immutability-eth-plugin/wallets
banner
vault list -output-curl-string immutability-eth-plugin/wallets

banner
echo "CREATE NEW ACCOUNT IN WALLET"
echo "vault write -f immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT0=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts

banner
echo "CHECK BALANCE FOR ACCOUNT IN WALLET"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance
banner
vault read -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance

banner
echo "CREATE SECOND NEW ACCOUNT IN WALLET"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts"
ACCOUNT1=$(vault write -f -field=address immutability-eth-plugin/wallets/test-wallet-2/accounts)
banner
vault write -output-curl-string -f immutability-eth-plugin/wallets/test-wallet-2/accounts

banner
echo "CHECK BALANCE FOR $ACCOUNT1 IN WALLET"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/balance"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/balance
banner
vault read -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/balance

banner
echo "TRANSFER FUNDS FROM $ACCOUNT0 TO $ACCOUNT1" 
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT
banner
vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/debit to=$ACCOUNT1 amount=$FUNDING_AMOUNT

banner
echo "CHECK NEW BALANCE FOR $ACCOUNT0 IN WALLET"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance
banner
vault read -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/balance

banner
echo "CHECK NEW BALANCE FOR $ACCOUNT1 IN WALLET"
echo "vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/balance"
vault read immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/balance
banner
vault read -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT1/balance

banner
echo "SIGN RAW TX FROM ACCOUNT"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to='$ACCOUNT1' data='hello' amount=1000000000000000"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to="$ACCOUNT1" data="hello" amount=1000000000000000
banner
vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to="$ACCOUNT1" data="hello" amount=1000000000000000

banner
echo "SIGN RAW TX FROM ACCOUNT WITH HEX ENCODING OF DATA"
echo "vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to='$ACCOUNT1' data='fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19' encoding='hex'  amount=1000000000000000"
vault write immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to="$ACCOUNT1" data="fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19" encoding="hex"  amount=1000000000000000
banner
vault write -output-curl-string immutability-eth-plugin/wallets/test-wallet-2/accounts/$ACCOUNT0/sign-tx to="$ACCOUNT1" data="fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19" encoding="hex"  amount=1000000000000000

banner
echo "EXPORT KEYSTORE FROM $ACCOUNT0"
echo "PASSPHRASE IS \"$PASSPHRASE\""
echo "vault immutability-eth-plugin/export/test-wallet-2/accounts/$ACCOUNT0 path=$(pwd) passphrase='$PASSPHRASE'"
vault write immutability-eth-plugin/export/test-wallet-2/accounts/$ACCOUNT0 path=$(pwd) passphrase="$PASSPHRASE"
banner
vault write -output-curl-string immutability-eth-plugin/export/test-wallet-2/accounts/$ACCOUNT0 path=$(pwd) passphrase="$PASSPHRASE"
banner
