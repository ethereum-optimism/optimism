#!/bin/bash
cmd="$@"
NODE_URL="http://localhost:8080"

function print_usage_and_exit {
    cat <<EOF
    $(basename $0) - health-check for deployer node
    Basic usage is to evoke the script.
    Global options:
        [--node <NODE_URL>]             Deployer node url [default: http://localhost:8080]
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

if [ ! -z "$NODE_URL" ]; then
    RETRIES=${RETRIES:-20}
    until $(curl --silent --fail \
        --output /dev/null \
        "$NODE_URL/addresses.json"); do
      sleep 1
      echo "Will wait $((RETRIES--)) more times for $NODE_URL to be up..."

      if [ "$RETRIES" -lt 0 ]; then
        echo "Timeout waiting for contract deployment"
        exit 1
      fi
    done
    echo "Contracts are deployed"
fi
