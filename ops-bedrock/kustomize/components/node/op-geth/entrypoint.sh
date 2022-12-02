#!/bin/sh
set -eu

apk add --no-cache curl

datadir=${OP_GETH_DATA_DIR:-/db}
verbosity=${GETH_VERBOSITY:-3}

# Check to see if a sequencer URL has been specified.
# To prevent MEV, Bedrock replicas send new transactions to a
# centralized sequencer endpoint. If you don't specify this,
# your replica will be unable to send transactions.

if [ -z "${OP_GETH_SEQUENCER_HTTP-}" ]; then
  sequencer_http=""
  echo "[WARNING] You haven't set OP_GETH_SEQUENCER_HTTP. Without this environment"
  echo "[WARNING] variable, you will not be able to send transactions."
else
  sequencer_http="$OP_GETH_SEQUENCER_HTTP"
fi

# Define a bunch of environment variables to configure Geth's behavior.

http_addr=${OP_GETH_HTTP_ADDR:-"0.0.0.0"}
http_corsdomain=${OP_GETH_HTTP_CORSDOMAIN:-"*"}
http_vhosts=${OP_GETH_HTTP_VHOSTS:-"*"}
http_port=${OP_GETH_HTTP_PORT:-"8545"}
http_api=${OP_GETH_HTTP_PORT:-"web3,debug,eth,txpool,net"}
gc_mode=${OP_GETH_GC_MODE:-"archive"}
ws_addr=${OP_GETH_WS_ADDR:-"0.0.0.0"}
ws_port=${OP_GETH_WS_PORT:-"8546"}
ws_origins=${OP_GETH_WS_ORIGINS:-"*"}
ws_api=${OP_GETH_HTTP_PORT:-"web3,debug,eth,txpool,net"}
authrpc_addr=${OP_GETH_AUTHRPC_ADDR:-"0.0.0.0"}
authrpc_port=${OP_GETH_AUTHRPC_PORT:-"8551"}
authrpc_vhosts=${OP_GETH_AUTHRPC_VHOSTS:-"*"}

# Check to see if we're running an archive node. Archival nodes are
# required to generate withdrawal proofs for all blocks except the
# highest 256.
#

if [ "$gc_mode" != "archive" ]; then
  echo "[WARNING] Warning! Setting OP_GETH_GC_MODE to something other than archive will"
  echo "[WARNING] prevent you from generating withdrawal proofs over RPC for blocks over"
  echo "[WARNING] 256 in the past."
fi

exec geth \
	--datadir="$datadir" \
	--verbosity="$verbosity" \
	--http \
	--http.addr="$http_addr" \
	--http.corsdomain="$http_corsdomain" \
	--http.vhosts="$http_vhosts" \
	--http.port="$http_port" \
	--http.api="$http_api" \
	--ws \
	--ws.addr="$ws_addr" \
	--ws.port="$ws_port" \
	--ws.origins="$ws_origins" \
	--ws.api="$ws_api" \
	--syncmode=full \
	--nodiscover \
	--maxpeers=0 \
	--gcmode="$gc_mode" \
	--rollup.disabletxpoolgossip=true \
	--rollup.sequencerhttp="$sequencer_http" \
	--authrpc.addr="$authrpc_addr" \
	--authrpc.port="$authrpc_port" \
	--authrpc.vhosts="$authrpc_vhosts" \
	--authrpc.jwtsecret=/etc/op-geth/jwt-secret.txt
	"$@"
