#!/bin/bash
cmd="$@"
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'
NODE_URL="http://localhost:8545"

function print_usage_and_exit {
    cat <<EOF
    $(basename $0) - health-check for L2 web3 node
    Basic usage is to evoke the script.
    Global options:
        [--node <NODE_URL>]             L2 web3 node url [default: http://localhost:8545]
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

RETRIES=${RETRIES:-30}
until $(curl --silent --fail \
    --output /dev/null \
    -H "Content-Type: application/json" \
    --data "$JSON" "$NODE_URL"); do
  sleep 1
  echo "Will wait $((RETRIES--)) more times for $NODE_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for layer two node at $NODE_URL"
    exit 1
  fi
done
echo "Connected to L2 Node at $NODE_URL"
