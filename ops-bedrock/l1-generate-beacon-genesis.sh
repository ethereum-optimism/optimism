#!/bin/sh

set -eu

echo "eth2-testnet-genesis path: $(which eth2-testnet-genesis)"

eth2-testnet-genesis deneb \
  --config=./beacon-data/config.yaml \
  --preset-phase0=minimal \
  --preset-altair=minimal \
  --preset-bellatrix=minimal \
  --preset-capella=minimal \
  --preset-deneb=minimal \
  --eth1-config=../.devnet/genesis-l1.json \
  --state-output=../.devnet/genesis-l1.ssz \
  --tranches-dir=../.devnet/tranches \
  --mnemonics=mnemonics.yaml \
  --eth1-withdrawal-address=0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --eth1-match-genesis-time
