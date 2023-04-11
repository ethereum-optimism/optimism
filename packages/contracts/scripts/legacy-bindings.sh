#!/bin/bash

OUTDIR="$1"

if [ ! -d "$OUTDIR" ]; then
    echo "Must pass output directory"
    exit 1
fi

CONTRACTS=("CanonicalTransactionChain" "StateCommitmentChain")
PKG=legacy_bindings

for contract in ${CONTRACTS[@]}; do
    TMPFILE=$(mktemp)
    npx hardhat inspect $contract bytecode > "$TMPFILE"
    ABI=$(npx hardhat inspect $contract abi)

    outfile="$OUTDIR/$contract.go"

    echo "$ABI" | abigen --abi - --pkg "$PKG" --bin "$TMPFILE" --type $contract --out "$outfile"

    rm "$TMPFILE"
done
