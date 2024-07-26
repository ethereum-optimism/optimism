#!/bin/bash
set -exu

# --allow-insecure-genesis-sync is required since we start from genesis, and it may be an old genesis
exec /usr/local/bin/lighthouse \
  bn \
  --datadir="/db" \
  --disable-peer-scoring \
  --disable-packet-filter \
  --enable-private-discovery \
  --staking \
  --http \
  --http-address=0.0.0.0 \
  --http-port=5052 \
  --validator-monitor-auto \
  --http-allow-origin='*' \
  --listen-address=0.0.0.0 \
  --port=9000 \
  --target-peers=0 \
  --testnet-dir=/genesis \
  --execution-endpoint="${LH_EXECUTION_ENDPOINT}" \
  --execution-jwt=/config/jwt-secret.txt \
  --allow-insecure-genesis-sync \
  --debug-level=info \
  "$@"
