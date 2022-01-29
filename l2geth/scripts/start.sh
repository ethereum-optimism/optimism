#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
REPO=$DIR/..

IS_VERIFIER=
ROLLUP_SYNC_SERVICE_ENABLE=true
DATADIR=$HOME/.ethereum
TARGET_GAS_LIMIT=15000000
ETH1_CTC_DEPLOYMENT_HEIGHT=12686738
ROLLUP_CLIENT_HTTP=http://localhost:7878
ROLLUP_POLL_INTERVAL=15s
ROLLUP_TIMESTAMP_REFRESH=15s
CACHE=1024
RPC_PORT=8545
WS_PORT=8546
VERBOSITY=3
ROLLUP_BACKEND=l2
CHAIN_ID=69
BLOCK_SIGNER_ADDRESS=0x00000398232E2064F896018496b4b44b3D62751F

USAGE="
Start the Sequencer or Verifier with most configuration pre-set.

CLI Arguments:
  -h|--help                              - help message
  -v|--verifier                          - start in verifier mode
  --datadir                              - data directory to use
  --chainid                              - layer two chain id to use, must match contracts on L1
  --rollup.clienthttp                    - rollup client http
  --rollup.pollinterval                  - polling interval for the rollup client
  --rollup.timestamprefresh              - timestamp refresh interval
  --cache                                - geth cache size
  --targetgaslimit                       - gas per block
"

while (( "$#" )); do
    case "$1" in
        -h|--help)
            echo "$USAGE"
            exit 0
            ;;
        -v|--verifier)
            IS_VERIFIER=true
            shift 1
            ;;
        --rollup.disablesyncservice)
            ROLLUP_SYNC_SERVICE_ENABLE=
            shift 1
            ;;
        --verbosity)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                VERBOSITY="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --datadir)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                DATADIR="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --rpcport)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                RPC_PORT="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --wsport)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                WS_PORT="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --eth1.ctcdeploymentheight)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ETH1_CTC_DEPLOYMENT_HEIGHT="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --rollup.clienthttp)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ROLLUP_CLIENT_HTTP="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --rollup.pollinterval)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ROLLUP_POLL_INTERVAL="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --rollup.timestamprefresh)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ROLLUP_TIMESTAMP_REFRESH="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --rollup.backend)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ROLLUP_BACKEND="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --cache)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                CACHE="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --targetgaslimit)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                TARGET_GASLIMIT="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        *)
            echo "Unknown argument $1" >&2
            shift
            ;;
    esac
done

cmd="$REPO/build/bin/geth"
if [[ ! -z "$ROLLUP_SYNC_SERVICE_ENABLE" ]]; then
    cmd="$cmd --eth1.syncservice"
fi
cmd="$cmd --datadir $DATADIR"
cmd="$cmd --eth1.ctcdeploymentheight $ETH1_CTC_DEPLOYMENT_HEIGHT"
cmd="$cmd --rollup.clienthttp $ROLLUP_CLIENT_HTTP"
cmd="$cmd --rollup.pollinterval $ROLLUP_POLL_INTERVAL"
cmd="$cmd --rollup.timestamprefresh $ROLLUP_TIMESTAMP_REFRESH"
cmd="$cmd --rollup.backend $ROLLUP_BACKEND"
cmd="$cmd --cache $CACHE"
cmd="$cmd --rpc"
cmd="$cmd --networkid $CHAIN_ID"
cmd="$cmd --rpcaddr 0.0.0.0"
cmd="$cmd --rpcport $RPC_PORT"
cmd="$cmd --rpcvhosts '*'"
cmd="$cmd --rpccorsdomain '*'"
cmd="$cmd --rpcvhosts '*'"
cmd="$cmd --ws"
cmd="$cmd --wsaddr 0.0.0.0"
cmd="$cmd --wsport $WS_PORT"
cmd="$cmd --wsorigins '*'"
cmd="$cmd --rpcapi eth,net,rollup,web3,debug,personal"
cmd="$cmd --gasprice 0"
cmd="$cmd --nousb"
cmd="$cmd --gcmode=archive"
cmd="$cmd --nodiscover"
cmd="$cmd --mine"
cmd="$cmd --password=$DATADIR/password"
cmd="$cmd --allow-insecure-unlock"
cmd="$cmd --unlock=$BLOCK_SIGNER_ADDRESS"
cmd="$cmd --miner.etherbase=$BLOCK_SIGNER_ADDRESS"
cmd="$cmd --txpool.pricelimit 0"

if [[ ! -z "$IS_VERIFIER" ]]; then
    cmd="$cmd --rollup.verifier"
fi
cmd="$cmd --verbosity=$VERBOSITY"

echo -e "Running:\nTARGET_GAS_LIMIT=$TARGET_GAS_LIMIT USING_OVM=true $cmd"
TARGET_GAS_LIMIT=$TARGET_GAS_LIMIT USING_OVM=true $cmd
