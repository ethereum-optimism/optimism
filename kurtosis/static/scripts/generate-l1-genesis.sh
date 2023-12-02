#!/bin/sh

set -euo pipefail

mkdir -p /execution-out
mkdir -p /consensus

cp /execution-in/genesis.json /tmp/genesis.json

# If /allocs-in exists, merge the geneis files
if [ -d "/allocs-in" ]; then
  echo "Merging genesis files."
  jq --slurpfile allocs /allocs-in/allocs.json '.alloc += $allocs[0]' /execution-in/genesis.json > /tmp/genesis.json
fi

prysmctl \
  testnet \
  generate-genesis \
  --fork=capella \
  --num-validators=64 \
  --output-ssz=/consensus/genesis.ssz \
  --chain-config-file=/config/config.yml \
  --geth-genesis-json-in=/tmp/genesis.json \
  --geth-genesis-json-out=/execution-out/genesis.json