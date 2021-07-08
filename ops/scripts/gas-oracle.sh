#!/bin/sh

RETRIES=${RETRIES:-40}

if [[ -z $GAS_PRICE_ORACLE_ETHEREUM_HTTP_URL ]]; then
    echo "Must set env GAS_PRICE_ORACLE_ETHEREUM_HTTP_URL"
    exit 1
fi

# waits for l2geth to be up
curl --fail \
    --show-error \
    --silent \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    --output /dev/null \
    $GAS_PRICE_ORACLE_ETHEREUM_HTTP_URL

exec gas-oracle "$@"
