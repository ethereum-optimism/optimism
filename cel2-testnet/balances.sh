#!/bin/bash
#
# Check L1 account balances and return with exit code 1 if at least one balance
# is below min_balance.
set -eo pipefail

SCRIPT_DIR=$(readlink -f $(dirname $0))
cd $SCRIPT_DIR
source .envrc

min_balance=4.0
exit_code=0

# SEQUENCER does not need any balance
for wallet in ADMIN PROPOSER BATCHER
do
    varname=${wallet}_ADDR
    printf "%-12s" ${wallet}:
    balance=$(cast balance -r $L1_RPC --ether ${!varname})
    echo $balance
    if [ $wallet != ADMIN ] && (( $(echo "$balance < $min_balance" | bc -l) ))
    then
        echo $wallet BALANCE LOW, send funds to ${!varname}!
        exit_code=1
    fi
done

exit $exit_code
