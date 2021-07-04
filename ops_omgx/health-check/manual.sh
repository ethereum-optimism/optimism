#!/bin/bash
cmd="$@"
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'
NODE_URL="http://localhost:9545"

function print_usage_and_exit {
    cat <<EOF
    $(basename $0) - Smoke test for EnyaLabs Optimism Integration Environment
    Basic usage is to evoke the script.
    Global options:
        [--node <NODE_URL>]             L1 web3 node url [default: http://localhost:9545]
        -h, --help                      This help :)
    Examples:
        $(basename $0) --node <NODE_URL>
EOF
    exit 2
}

function timestamp {
    local epoch=${1:-}

    if [[ $epoch == true ]] ; then
        date '+%s'
    else
        date '+%F %H:%M:%S'
    fi
}

function log_output {
    LOG_LEVEL="${1:-INFO}"
    echo "[$(timestamp)] $(basename ${0}) ${LOG_LEVEL}: ${@:2}" >&2
}

function error {
    log_output ERROR "${@}"
    exit 1
}

function getLatestBlock {
    JSON='{"jsonrpc":"2.0","id":0,"method":"eth_getBlockByNumber","params":["latest", true]}'
    RESULT=$(curl --silent --fail \
        -H "Content-Type: application/json" \
        --data "$JSON" "$NODE_URL");
    CHECK=$(python -c "import sys, json; print('0' if 'error' in json.loads('$RESULT') else '1')")
    ret=$?
    if [ $ret -ne 0 ]; 
    then
        echo "Error while getting latest block: $RESULT"
    else
        if [ $CHECK == "1" ]; then
            HASH=$(python -c "import sys, json; print(json.loads('$RESULT')['result']['hash'])")
            echo "Latest block hash: $HASH"
        else
            ERROR=$(python -c "import sys, json; print(json.loads('$RESULT')['error']['message'])")
            echo "Error while getting latest block: $ERROR"
        fi
    fi
}

function getGasPrice {
    JSON='{"jsonrpc":"2.0","id":0,"method":"eth_gasPrice","params":[]}'
    RESULT=$(curl --silent --fail \
        -H "Content-Type: application/json" \
        --data "$JSON" "$NODE_URL");
    CHECK=$(python -c "import sys, json; print('0' if 'error' in json.loads('$RESULT') else '1')")
    ret=$?
    if [ $ret -ne 0 ]; 
    then
        echo "Error while getting gas price: $RESULT"
    else
        if [ $CHECK == "1" ]; then
            PRICE=$(python -c "import sys, json; print(json.loads('$RESULT')['result'])")
            echo "Gas price: $(($PRICE)) gwei"
        else
            ERROR=$(python -c "import sys, json; print(json.loads('$RESULT')['error']['message'])")
            echo "Error while getting gas price: $ERROR"
        fi
    fi
}

function getChainId {
    JSON='{"jsonrpc":"2.0","id":0,"method":"eth_chainId","params":[]}'
    RESULT=$(curl --silent --fail \
        -H "Content-Type: application/json" \
        --data "$JSON" "$NODE_URL");
    CHECK=$(python -c "import sys, json; print('0' if 'error' in json.loads('$RESULT') else '1')")
    ret=$?
    if [ $ret -ne 0 ]; 
    then
        echo "Error while getting chain id: $RESULT"
    else
        if [ $CHECK == "1" ]; then
            CHAIN_ID=$(python -c "import sys, json; print(json.loads('$RESULT')['result'])")
            echo "Chain id: $(($CHAIN_ID))"
        else
            ERROR=$(python -c "import sys, json; print(json.loads('$RESULT')['error']['message'])")
            echo "Error while getting chain id: $ERROR"
        fi
    fi
}

if [[ $# -gt 0 ]]; then
    while [[ $# -gt 0 ]]; do
        case "${1}" in
            -h|--help)
                print_usage_and_exit
                ;;
            --node)
                NODE_URL="${2}"
                shift 2
                ;;
            --*)
                error "Unknown option ${1}"
                ;;
            *)
                error "Unknown sub-command ${1}"
                ;;
        esac
    done
else
    echo "Warning: command without option --node will use default value: $NODE_URL"
fi

PS3='=============Please enter your choice: '
options=("Get latest block" "Get gas price" "Get chain id" "Quit")
select opt in "${options[@]}"
do
    case $opt in
        "Get latest block")
            getLatestBlock
            ;;
        "Get gas price")
            getGasPrice
            ;;
        "Get chain id")
            getChainId
            ;;
        "Quit")
            break
            ;;
        *) echo "invalid option $REPLY";;
    esac
done