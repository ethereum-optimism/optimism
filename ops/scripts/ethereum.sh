#!/bin/sh

rm -rf data
# Set the following envs: CHAIN_ID, HTTP_PORT, WS_PORT, GASLIMIT, CHAIN_ADMIN_ADDRESS
# Creates accessible private key file of deployer
echo ${CHAIN_PRIVATE_KEY} > privkey.txt
echo ${CHAIN_PRIVATE_KEY}
echo "Wrote privkey"

# # Creates accessible geth password file for authentication
echo ${CHAIN_PASSWORD} > password.txt
echo ${CHAIN_PASSWORD}
echo "Wrote pass"
geth init --datadir data genesis.json

geth account import --datadir data --password password.txt privkey.txt

# start node
# geth --datadir data --networkid ${CHAIN_ID:-9090} --http --http.addr 0.0.0.0 --http.port ${HTTP_PORT:-9545} --http.vhosts "*" --ws --ws.addr 0.0.0.0 --ws.port ${WS_PORT:-9546} --nodiscover --unlock ${CHAIN_ADMIN_ADDRESS:-872425436273C68F0720E8b06C845A17A17853b9} --allow-insecure-unlock --mine --miner.gaslimit ${GASLIMIT:-32000000} --password password.txt
# cat ./genesis.json

geth  --datadir data \
      --networkid ${CHAIN_ID:-9090} \
      --mine \
      --miner.threads 2 \
      --miner.gaslimit ${GASLIMIT:-60000000} \
      --http --http.addr 0.0.0.0 \
      --http.port ${HTTP_PORT:-8545} \
      --http.vhosts "*" \
      --http.api admin,eth,miner,net,txpool,personal,web3,debug \
      --ws --ws.addr 0.0.0.0 \
      --ws.port ${WS_PORT:-8546} \
      --unlock ${CHAIN_ADMIN_ADDRESS} \
      --allow-insecure-unlock \
      --password password.txt \
      --verbosity 4 \
      --nodiscover \
      "$@"

      # --miner.gasprice 0 \

      # --metrics \
      # --metrics.influxdb \
      # --metrics.influxdb.endpoint "http://0.0.0.0:8087" \
      # --metrics.influxdb.database "l1_chain"
      # --metrics.influxdb.username "admin" \
      # --metrics.influxdb.password "ethereum"

#test
# sleep 5
