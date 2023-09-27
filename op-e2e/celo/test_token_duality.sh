#!/bin/bash
#shellcheck disable=SC2086
set -eo pipefail

source shared.sh

# Send token and check balance
balance_before=$(cast balance 0x000000000000000000000000000000000000dEaD)
cast send --private-key $ACC_PRIVKEY $TOKEN_ADDR 'transfer(address to, uint256 value) returns (bool)' 0x000000000000000000000000000000000000dEaD 100
balance_after=$(cast balance 0x000000000000000000000000000000000000dEaD)
echo "Balance change: $balance_before -> $balance_after"
[[ $((balance_before + 100)) -eq $balance_after ]] || (echo "Balance did not change as expected"; exit 1)
