#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
REPO=$DIR/..

IS_VERIFIER=
ROLLUP_SYNC_SERVICE_ENABLE=true
DATADIR=$HOME/.ethereum
TARGET_GAS_LIMIT=9000000
CHAIN_ID=10
ETH1_CTC_DEPLOYMENT_HEIGHT=12410807
ETH1_L1_GATEWAY_ADDRESS=0xe681F80966a8b1fFadECf8068bD6F99034791c95
ETH1_L1_CROSS_DOMAIN_MESSENGER_ADDRESS=0x902e5fF5A99C4eC1C21bbab089fdabE32EF0A5DF
ADDRESS_MANAGER_OWNER_ADDRESS=0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A
ROLLUP_STATE_DUMP_PATH=https://storage.googleapis.com/optimism/mainnet/4.json
ROLLUP_CLIENT_HTTP=http://localhost:7878
ROLLUP_POLL_INTERVAL=15s
ROLLUP_TIMESTAMP_REFRESH=3m
CACHE=1024
RPC_PORT=8545
WS_PORT=8546
VERBOSITY=3
ROLLUP_BACKEND=l1

USAGE="
Start the Sequencer or Verifier with most configuration pre-set.

CLI Arguments:
  -h|--help                              - help message
  -v|--verifier                          - start in verifier mode
  --datadir                              - data directory to use
  --chainid                              - layer two chain id to use, must match contracts on L1
  --eth1.chainid                         - eth1 chain id
  --eth1.ctcdeploymentheight             - eth1 ctc deploy height
  --eth1.l1crossdomainmessengeraddress   - eth1 l1 xdomain messenger address
  --rollup.statedumppath                 - http path to the initial state dump
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
        --chainid)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                CHAIN_ID="$2"
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
        --eth1.l1gatewayaddress)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ETH1_L1_GATEWAY_ADDRESS="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --eth1.l1crossdomainmessengeraddress)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ETH1_L1_CROSS_DOMAIN_MESSENGER_ADDRESS="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --eth1.l1ethgatewayaddress)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ETH1_L1_ETH_GATEWAY_ADDRESS="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --eth1.ctcdeploymentheight)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ADDRESS_MANAGER_OWNER_ADDRESS="$2"
                shift 2
            else
                echo "Error: Argument for $1 is missing" >&2
                exit 1
            fi
            ;;
        --rollup.statedumppath)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ROLLUP_STATE_DUMP_PATH="$2"
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
        --rollup.addressmanagerowneraddress)
            if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
                ADDRESS_MANAGER_OWNER_ADDRESS="$2"
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
cmd="$cmd --eth1.l1crossdomainmessengeraddress $ETH1_L1_CROSS_DOMAIN_MESSENGER_ADDRESS"
cmd="$cmd --rollup.addressmanagerowneraddress $ADDRESS_MANAGER_OWNER_ADDRESS"
cmd="$cmd --rollup.statedumppath $ROLLUP_STATE_DUMP_PATH"
cmd="$cmd --eth1.ctcdeploymentheight $ETH1_CTC_DEPLOYMENT_HEIGHT"
cmd="$cmd --eth1.l1ethgatewayaddress $ETH1_L1_GATEWAY_ADDRESS"
cmd="$cmd --rollup.clienthttp $ROLLUP_CLIENT_HTTP"
cmd="$cmd --rollup.pollinterval $ROLLUP_POLL_INTERVAL"
cmd="$cmd --rollup.timestamprefresh $ROLLUP_TIMESTAMP_REFRESH"
cmd="$cmd --rollup.backend $ROLLUP_BACKEND"
cmd="$cmd --cache $CACHE"
cmd="$cmd --rpc"
cmd="$cmd --dev"
cmd="$cmd --chainid $CHAIN_ID"
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
cmd="$cmd --rpcapi 'eth,net,rollup,web3,debug'"
cmd="$cmd --gasprice 0"
cmd="$cmd --nousb"
cmd="$cmd --gcmode=archive"
cmd="$cmd --ipcdisable"
if [[ ! -z "$IS_VERIFIER" ]]; then
    cmd="$cmd --rollup.verifier"
fi
cmd="$cmd --verbosity=$VERBOSITY"

echo -e "Running:\nTARGET_GAS_LIMIT=$TARGET_GAS_LIMIT USING_OVM=true $cmd"
eval env TARGET_GAS_LIMIT=$TARGET_GAS_LIMIT USING_OVM=true $cmd
