#!/bin/sh
set -e

# Set the default ports if not provided.
OP_HTTP_PORT=${OP_HTTP_PORT:-8546}
OP_ENGINE_PORT=$${OP_ENGINE_PORT:-8552}

# Initialize database if not already done.
if [ ! -f /data/geth/chaindata/CURRENT ]; then
  echo "Initializing op-geth database..."
  geth init --datadir=/data --state.scheme=path /l2-genesis-devnet.json
  echo "op-geth initialization completed"
else
  echo "op-geth database already initialized, skipping..."
fi

# Start op-geth with the specified configuration
exec geth \
    --datadir=/data \
    --networkid=1 \
    --http \
    --http.addr=0.0.0.0 \
    --http.port=${OP_HTTP_PORT} \
    --http.api=eth,net,web3,debug,admin,txpool \
    --http.vhosts=* \
    --http.corsdomain=* \
    --authrpc.addr=0.0.0.0 \
    --authrpc.port=${OP_ENGINE_PORT} \
    --authrpc.vhosts=* \
    --authrpc.jwtsecret=/config/jwt.txt \
    --rollup.disabletxpoolgossip=true \
    --rollup.halt=major \
    --nodiscover
