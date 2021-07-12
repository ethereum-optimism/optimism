#!/bin/sh


function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

source /vault/scripts/smoke.env.sh
OVM="OVM-wallet"
batch=['0x1111111111111111111111111111111111111111111111111111111111111111', '0x1111111111111111111111111111111111111111111111111111111111111111']

banner
echo "CONFIGURE MOUNT"
echo "vault write -format=json immutability-eth-plugin/config  rpc_url='$RPC_URL' chain_id='$CHAIN_ID' rpc_l2_url='$RPC_L2_URL' chain_l2_id='$CHAIN_L2_ID'"
vault write -format=json immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"
check_result $? 0
banner
#vault write  -output-curl-string immutability-eth-plugin/config rpc_url="$RPC_URL" chain_id="$CHAIN_ID" rpc_l2_url="$RPC_L2_URL" chain_l2_id="$CHAIN_L2_ID"

banner
echo "CREATE WALLET WITH MNEMONIC FOR PLASMA DEPLOYER"
echo "vault write -format=json immutability-eth-plugin/wallets/$OVM mnemonic='$MNEMONIC'"
vault write -format=json immutability-eth-plugin/wallets/$OVM mnemonic="$MNEMONIC"
check_result $? 0
banner
#vault write  -output-curl-string immutability-eth-plugin/wallets/$OVM mnemonic="$MNEMONIC"

banner
echo "CREATE DEPLOYER ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/$OVM/accounts"
DEPLOYER=$(vault write -f -field=address immutability-eth-plugin/wallets/$OVM/accounts)
check_result $? 0
echo "DEPLOYER=$DEPLOYER"
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/$OVM/accounts

banner
echo "CREATE MAINTAINER ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/$OVM/accounts"
MAINTAINER=$(vault write -f -field=address immutability-eth-plugin/wallets/$OVM/accounts)
check_result $? 0
echo "MAINTAINER=$MAINTAINER"
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/$OVM/accounts

banner
echo "CREATE AUTHORITY ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/$OVM/accounts"
ORIGINAL_AUTHORITY=$(vault write -f -field=address immutability-eth-plugin/wallets/$OVM/accounts)
check_result $? 0
echo "ORIGINAL_AUTHORITY=$ORIGINAL_AUTHORITY"
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/$OVM/accounts

banner
echo "CREATE UNAUTHORIZED ACCOUNT IN WALLET"
echo "vault write -format=json -f immutability-eth-plugin/wallets/$OVM/accounts"
UNAUTHORIZED=$(vault write -f -field=address immutability-eth-plugin/wallets/$OVM/accounts)
check_result $? 0
echo "UNAUTHORIZED=$UNAUTHORIZED"
banner
#vault write -format=json -f -output-curl-string immutability-eth-plugin/wallets/$OVM/accounts
banner

banner
banner
banner
echo "Append State Batch tests"
echo "*** SHOULD FAIL! ***" 
echo "UNAUTHORIZED SUBMISSION OF AppendStateBatch BY $UNAUTHORIZED" 
echo "vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$UNAUTHORIZED/ovm/appendStateBatch batch="0x1111111111111111111111111111111111111111111111111111111111111111" batch="0x1111111111111111111111111111111111111111111111111111111111111111" should_start_at_element=0 nonce=0 gas_price=$GAS_PRICE_HIGH contract=$PLASMA_CONTRACT"
vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$UNAUTHORIZED/ovm/appendStateBatch batch="0x1111111111111111111111111111111111111111111111111111111111111111" batch="0x1111111111111111111111111111111111111111111111111111111111111111" should_start_at_element=0 nonce=0 gas_price=$GAS_PRICE_HIGH contract=$PLASMA_CONTRACT
check_result $? 2
banner
#vault write  -output-curl-string immutability-eth-plugin/wallets/$OVM/accounts/$UNAUTHORIZED/ovm/appendStateBatch nonce=0 gas_price=$GAS_PRICE_HIGH batch=$batch should_start_at_element=0 contract=$PLASMA_CONTRACT

# banner
# echo "*** SHOULD SUCCEED ***" 
# echo "AUTHORIZED SUBMISSION OF BATCH BY $ORIGINAL_AUTHORITY" 
# echo "vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$ORIGINAL_AUTHORITY/ovm/appendStateBatch nonce=0 gas_price=$GAS_PRICE_HIGH batch=$batch should_start_at_element=0 contract=$PLASMA_CONTRACT"
# vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$ORIGINAL_AUTHORITY/ovm/appendStateBatch nonce=0 gas_price=$GAS_PRICE_HIGH batch=$batch should_start_at_element=0 contract=$PLASMA_CONTRACT
# check_result $? 0
# banner
# #vault write  -output-curl-string immutability-eth-plugin/wallets/$OVM/accounts/$ORIGINAL_AUTHORITY/ovm/appendStateBatch nonce=0 gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT

# banner
# echo "*** SHOULD FAIL ***" 
# echo "AUTHORIZED SUBMISSION OF BAD BLOCK BY $ORIGINAL_AUTHORITY" 
# echo "vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BAD_BLOCK_ROOT contract=$PLASMA_CONTRACT"
# vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BAD_BLOCK_ROOT contract=$PLASMA_CONTRACT
# check_result $? 2
# banner
# #vault write  -output-curl-string immutability-eth-plugin/wallets/$OVM/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BAD_BLOCK_ROOT contract=$PLASMA_CONTRACT

# banner
# echo "*** SHOULD SUCCEED ***" 
# echo "AUTHORIZED SUBMISSION OF BLOCK BY $ORIGINAL_AUTHORITY - USER SUPPLIED GAS PRICE" 
# echo "vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT"
# vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT
# check_result $? 0
# banner
# #vault write  -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT

# banner
# echo "*** SHOULD FAIL! ***" 
# echo "AUTHORIZED SUBMISSION OF BLOCK BY $ORIGINAL_AUTHORITY - USER SUPPLIED GAS PRICE WITHOUT NONCE" 
# echo "vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT"
# vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT
# check_result $? 2
# banner
# #vault write  -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock gas_price=$GAS_PRICE_HIGH block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT

# banner
# echo "*** SHOULD FAIL! ***" 
# echo "AUTHORIZED SUBMISSION OF BLOCK BY $ORIGINAL_AUTHORITY - USER SUPPLIED NONCE WITHOUT GAS PRICE" 
# echo "vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT"
# vault write -format=json immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT
# check_result $? 2
# banner
# #vault write  -output-curl-string immutability-eth-plugin/wallets/plasma-deployer/accounts/$ORIGINAL_AUTHORITY/plasma/submitBlock nonce=1 block_root=$BLOCK_ROOT contract=$PLASMA_CONTRACT

banner
banner
banner
echo "Append Sequencer Batch tests"
echo "*** SHOULD FAIL! ***" 
echo "UNAUTHORIZED SUBMISSION OF AppendSequencerBatch BY $UNAUTHORIZED" 
echo "vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$UNAUTHORIZED/ovm/appendSequencerBatch should_start_at_element=10 total_elements_to_append=1 contexts="{\"num_sequenced_transactions\": 2, \"num_subsequent_queue_transactions\": 1, \"timestamp\": 100, \"block_number\": 200}" transactions="0x45423400000011" transactions="0x45423400000012" nonce=0 gas_price=$GAS_PRICE_HIGH contract=$PLASMA_CONTRACT"
vault write -format=json immutability-eth-plugin/wallets/$OVM/accounts/$UNAUTHORIZED/ovm/appendSequencerBatch should_start_at_element=10 total_elements_to_append=1 contexts="{\"num_sequenced_transactions\": 2, \"num_subsequent_queue_transactions\": 1, \"timestamp\": 100, \"block_number\": 200}" transactions="0x45423400000011" transactions="0x45423400000012" nonce=0 gas_price=$GAS_PRICE_HIGH contract=$PLASMA_CONTRACT
check_result $? 2
banner