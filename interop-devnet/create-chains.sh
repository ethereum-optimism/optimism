#!/bin/bash

set -eu

# Run this with workdir set as root of the repo
if [ -f "versions.json" ]; then
    echo "Running create-chains script."
else
    echo "Cannot run create-chains script, must be in root of repository, but currently in:"
    echo "$(pwd)"
    exit 1
fi

# Check if already created
if [ -d ".devnet-interop" ]; then
    echo "Already created chains."
    exit 1
else
    echo "Creating new interop devnet chain configs"
fi

mkdir ".devnet-interop"

export CONTRACTS_ARTIFACTS_DIR="../packages/contracts-bedrock"

cd "../.devnet-interop/"

# deploy/     -- read only
#   l1/
#     dev.toml
#   superchain/
#     dev.toml
#   l2/
#     a.toml
#     b.toml
# out/
#   l1/
#     dev/
#       l1-addresses.json
#       genesis.json
#       meta.json
#   superchain/
#     dev/
#       l1-addresses.json
#       meta.json
#   l2/
#     a/
#       l1-addresses.json
#       rollup.json
#       genesis.json
#       meta.json
#     b/
#       l1-addresses.json
#       rollup.json
#       genesis.json
#       meta.json

go run ../op-node dev

# create L1 CL genesis
eth2-testnet-genesis deneb \
  --config=./beacon-data/config.yaml \
  --preset-phase0=minimal \
  --preset-altair=minimal \
  --preset-bellatrix=minimal \
  --preset-capella=minimal \
  --preset-deneb=minimal \
  --eth1-config=../.devnet-interop/out/l1/genesis.json \
  --state-output=../.devnet-interop/out/l1/beaconstate.ssz \
  --tranches-dir=../.devnet-interop/out/l1/tranches \
  --mnemonics=mnemonics.yaml \
  --eth1-withdrawal-address=0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --eth1-match-genesis-time

# create L2 A genesis + rollup config
# TODO this needs to be refactored
# Also the L1 RPC part should be static, to avoid temporary L1 node going up.
#go run ./op-node/cmd/main.go genesis l2 \
#  --l1-rpc http://localhost:8545 \
#  --deploy-config devnet_config_path \
#  --l2-allocs ./.devnet-interop/allocs-l2-a.json \
#  --l1-deployments ./.devnet-interop/l1-deployments-a.json \
#  --outfile.l2 ./.devnet-interop/genesis-l2-a.json \
#  --outfile.rollup ./.devnet-interop/rollup-a.json

# create L2 B genesis + rollup config
# TODO repeat for L2 B
