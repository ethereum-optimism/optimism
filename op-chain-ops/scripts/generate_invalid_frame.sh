#!/bin/bash

set -euo pipefail

id=$(openssl rand -hex 16)
number="0000"
length="00000000"
is_last="00"


out="0x00$id$number$length$is_last"
echo $id
echo $out
# Hard coded for the devnet
cast send 0xff00000000000000000000000000000000000901 $out --mnemonic="test test test test test test test test test test test junk" --mnemonic-derivation-path="m/44'/60'/0'/0/2" --rpc-url=http://localhost:8545
