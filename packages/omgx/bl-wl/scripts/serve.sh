#!/bin/bash

set -e
RETRIES=${RETRIES:-70}

echo "Waiting for OMGX Deployer at $OMGX_URL"

until $(curl --silent --fail \
	--show-error --retry-connrefused \
	$OMGX_URL); do
  sleep 2
  echo "Will wait $((RETRIES--)) more times for OMGX Deployer $OMGX_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for OMGX Deployer at $OMGX_URL"
    exit 1
  fi
done

echo "Connected to OMGX Deployer at $OMGX_URL - now fetching addresses"

# get the addresses from the URL provided
ADDRESSES=$(curl --fail --show-error --silent $OMGX_URL)

echo $ADDRESSES

# set the env
export L1LIQPOOL=$(echo $ADDRESSES | jq -r '.L1LiquidityPool')
export L1M=$(echo $ADDRESSES | jq -r '.L1Message')
echo '["'$L1LIQPOOL'", "'$L1M'"]' > dist/dumps/whitelist.json

# serve the addresses
exec ./bin/serve_dump.sh
