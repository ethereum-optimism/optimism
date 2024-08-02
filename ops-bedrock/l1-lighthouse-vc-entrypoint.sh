#!/bin/bash
set -exu

# Copy the keys and secrets into place:
# We don't mount directly into place, to prevent the container from writing lock-files / slashing-db / etc.
# back into the host, which could affect future fresh devnet runs if not cleaned up.
cp -r /validator_setup/validators /db/validators
cp -r /validator_setup/secrets /db/secrets

exec /usr/local/bin/lighthouse \
  vc \
  --datadir="/db" \
  --beacon-nodes="${LH_BEACON_NODES}" \
  --testnet-dir=/genesis \
  --init-slashing-protection \
  --suggested-fee-recipient="0xff00000000000000000000000000000000c0ffee" \
  "$@"
