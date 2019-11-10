#!/bin/sh


function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

source /home/vault/scripts/smoke.env.sh

CONTRACT_SAFE_MATH="SafeMath"
CONTRACT_OWNED="Owned"
CONTRACT_FIXED_SUPPLY_TOKEN="FixedSupplyToken"
BIN_FILE=".bin"
ABI_FILE=".abi"

banner
echo "CONFIGURE MOUNT"
echo "vault write immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID'"
vault write immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"
banner
vault write -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID"

banner
echo "CREATE WALLET WITH MNEMONIC FOR DEPLOYING ERC20 TOKEN"
echo "vault write immutability-eth-plugin/wallets/erc20-deployer mnemonic='$MNEMONIC'"
vault write immutability-eth-plugin/wallets/erc20-deployer mnemonic="$MNEMONIC"
banner
vault write -output-curl-string immutability-eth-plugin/wallets/erc20-deployer mnemonic="$MNEMONIC"

banner
echo "CREATE WALLET FOR TESTING RECEIPT OF TOKENS"
echo "vault write -f immutability-eth-plugin/wallets/test-recipient"
vault write -f immutability-eth-plugin/wallets/test-recipient
banner
vault write -output-curl-string -f immutability-eth-plugin/wallets/test-recipient

banner
echo "CREATE NEW ACCOUNT IN DEPLOYER WALLET"
echo "vault write -f immutability-eth-plugin/wallets/erc20-deployer/accounts"
DEPLOYER_ACCOUNT=$(vault write -f -field=address immutability-eth-plugin/wallets/erc20-deployer/accounts)
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/erc20-deployer/accounts

banner
echo "CREATE NEW ACCOUNT IN RECIPIENT WALLET"
echo "vault write -f immutability-eth-plugin/wallets/test-recipient/accounts"
RECIPIENT_ACCOUNT=$(vault write -f -field=address immutability-eth-plugin/wallets/test-recipient/accounts)
banner
vault write -f -output-curl-string immutability-eth-plugin/wallets/test-recipient/accounts

banner
echo "DEPLOY CONTRACT $CONTRACT_FIXED_SUPPLY_TOKEN FROM $DEPLOYER_ACCOUNT"
echo "vault write immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$BIN_FILE"
ERC20_CONTRACT_ADDRESS=$(vault write -field=contract immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$BIN_FILE)
banner
vault write -output-curl-string immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_FIXED_SUPPLY_TOKEN$BIN_FILE

banner
echo "DEPLOY CONTRACT $CONTRACT_OWNED FROM $DEPLOYER_ACCOUNT"
echo "vault write immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_OWNED$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_OWNED$BIN_FILE"
vault write immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_OWNED$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_OWNED$BIN_FILE
banner
vault write -output-curl-string immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_OWNED$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_OWNED$BIN_FILE

banner
echo "DEPLOY CONTRACT $CONTRACT_SAFE_MATH FROM $DEPLOYER_ACCOUNT"
echo "vault write immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$BIN_FILE"
vault write immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$BIN_FILE
banner
vault write -output-curl-string immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/deploy abi=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$ABI_FILE bin=@$CONTRACTS_PATH$CONTRACT_SAFE_MATH$BIN_FILE

banner
echo "READ ERC20 TOKEN TOTAL SUPPLY"
echo "vault read immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/totalSupply contract='$ERC20_CONTRACT_ADDRESS'"
vault read immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/totalSupply contract="$ERC20_CONTRACT_ADDRESS"
banner
vault write -output-curl-string immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/totalSupply contract="$ERC20_CONTRACT_ADDRESS"

banner
echo "READ ERC20 TOKEN BALANCE AT $DEPLOYER_ACCOUNT"
echo "vault read immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/balanceOf contract='$ERC20_CONTRACT_ADDRESS'"
vault read immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/balanceOf contract="$ERC20_CONTRACT_ADDRESS"
banner
vault write -output-curl-string immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/balanceOf contract="$ERC20_CONTRACT_ADDRESS"

banner
echo "TRANSFER TOKEN FROM ACCOUNT"
echo "vault write immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/transfer contract='$ERC20_CONTRACT_ADDRESS'  to='$RECIPIENT_ACCOUNT' tokens=23"
vault write immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/transfer contract="$ERC20_CONTRACT_ADDRESS" to="$RECIPIENT_ACCOUNT" tokens=23
banner
vault write -output-curl-string immutability-eth-plugin/wallets/erc20-deployer/accounts/$DEPLOYER_ACCOUNT/erc-20/transfer contract="$ERC20_CONTRACT_ADDRESS" to="$RECIPIENT_ACCOUNT" tokens=23

banner
echo "READ ERC20 TOKEN BALANCE AT $RECIPIENT_ACCOUNT"
echo "vault read immutability-eth-plugin/wallets/test-recipient/accounts/$RECIPIENT_ACCOUNT/erc-20/balanceOf contract='$ERC20_CONTRACT_ADDRESS'"
vault read immutability-eth-plugin/wallets/test-recipient/accounts/$RECIPIENT_ACCOUNT/erc-20/balanceOf contract="$ERC20_CONTRACT_ADDRESS"
banner
vault write -output-curl-string immutability-eth-plugin/wallets/test-recipient/accounts/$RECIPIENT_ACCOUNT/erc-20/balanceOf contract="$ERC20_CONTRACT_ADDRESS"
