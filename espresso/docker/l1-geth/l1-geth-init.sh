#!/bin/bash
set -e

# Set the default port if not provided.
L1_HTTP_PORT=${L1_HTTP_PORT:-8545}
L1_ENGINE_PORT=${L1_ENGINE_PORT:-8551}

# Initialize database if not already done.
if [ ! -f /data/geth/chaindata/CURRENT ]; then
  echo "Initializing L1 Geth database..."
  rm -rf /data/geth || true
  geth --datadir /data --gcmode=archive --state.scheme=hash init /config/genesis.json
  echo "L1 Geth initialization completed"
else
  echo "L1 Geth database already initialized, skipping..."
fi

# Start Geth with the specified configuration.
exec geth \
  --datadir /data/geth \
  --http \
  --http.addr=0.0.0.0 \
  --http.api=eth,net,web3,admin,engine,miner \
  --http.port=${L1_HTTP_PORT} \
  --http.vhosts=* \
  --http.corsdomain=* \
  --authrpc.addr=0.0.0.0 \
  --authrpc.port=${L1_ENGINE_PORT} \
  --authrpc.vhosts=* \
  --authrpc.jwtsecret=/config/jwt.txt \
  --nodiscover \
  --maxpeers 0 \
  --networkid ${L1_CHAIN_ID} \
  --syncmode=full \
  --gcmode=archive
