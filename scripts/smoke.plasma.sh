#!/bin/sh


function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

source /home/vault/scripts/smoke.env.sh

BLOCK_ROOT="1234qweradgf1234qweradgf"

banner
echo "CONFIGURE MOUNT"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"
banner
vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"

banner
echo "CREATE WALLET WITH MNEMONIC FOR PLASMA DEPLOYER"
echo "vault write -format=json immutability-eth-plugin/wallets/plasma-deployer mnemonic='$MNEMONIC'"
vault write -format=json immutability-eth-plugin/wallets/plasma-deployer mnemonic="$MNEMONIC"
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/plasma-deployer mnemonic="$MNEMONIC"

banner
echo "CREATE DEPLOYER ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/plasma-deployer/accounts"
DEPLOYER=$(vault write -f -field=address immutability-eth-plugin/wallets/plasma-deployer/accounts)
echo "DEPLOYER=$DEPLOYER"
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts

banner
echo "CREATE MAINTAINER ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/plasma-deployer/accounts"
MAINTAINER=$(vault write -f -field=address immutability-eth-plugin/wallets/plasma-deployer/accounts)
echo "MAINTAINER=$MAINTAINER"
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts

banner
echo "CREATE AUTHORITY ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/plasma-deployer/accounts"
ORIGINAL_AUTHORITY=$(vault write -f -field=address immutability-eth-plugin/wallets/plasma-deployer/accounts)
echo "ORIGINAL_AUTHORITY=$ORIGINAL_AUTHORITY"
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts

banner
echo "CREATE UNAUTHORIZED ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/plasma-deployer/accounts"
UNAUTHORIZED=$(vault write -f -field=address immutability-eth-plugin/wallets/plasma-deployer/accounts)
echo "UNAUTHORIZED=$UNAUTHORIZED"
banner
vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts
banner
echo "*** SHOULD FAIL! ***" 
echo "UNAUTHORIZED SUBMISSION OF BLOCK BY $UNAUTHORIZED" 
echo "vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$UNAUTHORIZED/plasma/submitBlock nonce=0 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT"
vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$UNAUTHORIZED/plasma/submitBlock nonce=0 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts/$UNAUTHORIZED/plasma/submitBlock nonce=0 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT

banner
echo "*** SHOULD SUCCEED ***" 
echo "AUTHORIZED SUBMISSION OF BLOCK BY $ORIGINAL_AUTHORITY" 
echo "vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=0 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT"
vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=0 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=0 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT

banner
echo "*** SHOULD SUCCEED ***" 
echo "AUTHORIZED SUBMISSION OF BLOCK BY $ORIGINAL_AUTHORITY - USER SUPPLIED GAS PRICE" 
echo "vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT"
vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT
banner
vault write  -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT
