#!/bin/bash
set -e

DATADIR=/db

COMMON_FLAGS=" \
  --chain dev \
  --datadir ${DATADIR} \
  --log.console.verbosity dbug \
  "

ERIGON_FLAGS=" \
  ${COMMON_FLAGS} \
  --ws \
  --mine \
  --miner.etherbase=0x123463a4B065722E99115D6c222f267d9cABb524 \
  --miner.sigfile ${DATADIR}/nodekey \
  --http.port 8545 \
  --http.addr 0.0.0.0 \
  --http.vhosts l2,localhost \
  --http.corsdomain '*' \
  --http.api eth,debug,net,engine,erigon,web3 \
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
  --authrpc.jwtsecret /config/test-jwt-secret.txt \
  --networkid 901 \
  "

RPC_FLAGS=" \
  ${COMMON_FLAGS} \
  --http.port 8545 \
  --http.addr 0.0.0.0 \
  --http.vhosts l2,localhost \
  --http.corsdomain '*' \
  --http.api eth,debug,net,erigon,web3 \
"

if [ ! -f ${DATADIR}/init_done ] ; then
  echo "Chain init"
  erigon ${COMMON_FLAGS} init /genesis.json
  echo "Creating keyfile"
  echo "2e0834786285daccd064ca17f1654f67b4aef298acbb82cef9ec422fb4975622" > ${DATADIR}/nodekey
  touch ${DATADIR}/init_done
  echo "Init completed"
  echo
fi

echo "---------------"
echo ${ERIGON_FLAGS}
echo "--------------"
#while true ; do
	erigon ${ERIGON_FLAGS}
        echo
        echo ***** EXITED WITH STATUS $? *****
        echo
#done
#sleep 5
#rpcdaemon ${RPC_FLAGS}

  

 
