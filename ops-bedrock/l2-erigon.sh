#!/bin/bash
set -e

DATADIR=/db
VERBOSITY=${VERBOSITY:-3}
CHAIN_NAME=${CHAIN_NAME:-dev}
BLOCK_SIGNER_PRIVATE_KEY="2e0834786285daccd064ca17f1654f67b4aef298acbb82cef9ec422fb4975622"

COMMON_FLAGS=" \
  --chain ${CHAIN_NAME} \
  --datadir ${DATADIR} \
  --log.console.verbosity ${VERBOSITY} \
  "

ERIGON_FLAGS=" \
  ${COMMON_FLAGS} \
  --http.port=8545 \
  --http.addr=0.0.0.0 \
  --http.vhosts=* \
  --http.corsdomain=* \
  --http.api=eth,debug,net,engine,erigon,web3 \
  --ws \
  --ws.port=8545 \
  --private.api.addr=0.0.0.0:9090 \
  --allow-insecure-unlock \
  --metrics \
  --metrics.addr=0.0.0.0 \
  --metrics.port=6060 \
  --pprof \
  --pprof.addr=0.0.0.0 \
  --pprof.port=6061 \
  --authrpc.addr=0.0.0.0 \
  --authrpc.port=8551 \
  --authrpc.vhosts=* \
  --authrpc.jwtsecret /config/jwt-secret.txt \
  --db.size.limit=8TB \
  "

if [ -z "$CHAIN_NAME" ]; then
  echo "CHAIN_NAME must be set to init chaindata"
  exit 1
fi

if [ -n "$CHAIN_ID" ]; then
  ERIGON_FLAGS="${ERIGON_FLAGS} --networkid ${CHAIN_ID}"
fi

if [ "$CHAIN_NAME" == "dev" ] && [ -z "$CHAIN_ID" ]; then
  echo "CHAIN_ID must be set for dev chain"
  exit 1
fi

if [ ! -d "${DATADIR}/chaindata" ] ; then
	echo "${DATADIR}/chaindata  missing, running init"
  # shellcheck disable=SC2086
  erigon ${COMMON_FLAGS} init /config/genesis-l2.json
  echo "Creating keyfile"
  echo ${BLOCK_SIGNER_PRIVATE_KEY} > ${DATADIR}/nodekey
  echo "Init completed"
  echo
else
  echo "${DATADIR}/chaindata found, skipping init"
fi

echo "---------------"
# shellcheck disable=SC2086
echo ${ERIGON_FLAGS} "$@"
echo "--------------"

# shellcheck disable=SC2086
exec erigon ${ERIGON_FLAGS} "$@"
echo
echo ***** EXITED WITH STATUS $? *****
echo
