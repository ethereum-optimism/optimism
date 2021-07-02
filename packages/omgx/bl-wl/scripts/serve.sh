#!/bin/bash
set -e

#!/bin/bash

set -e

RETRIES=${RETRIES:-60}

# get the addrs from the URL provided
ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $OMGX_URL)
# set the env
export L1LIQPOOL=$(echo $ADDRESSES | jq -r '.L1LiquidityPool')
export L1M=$(echo $ADDRESSES | jq -r '.L1Message')
echo '["'$L1LIQPOOL'", "'$L1M'"]' > dist/dumps/whitelist.json

# serve the addrs and the state dump
exec ./bin/serve_dump.sh
