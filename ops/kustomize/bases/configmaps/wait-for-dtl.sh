#!/bin/bash
set -eou
if [[ -z $ROLLUP_CLIENT_HTTP ]]; then
    echo "Must pass ROLLUP_CLIENT_HTTP"
    exit 1
fi
echo "Waiting for DTL"
curl \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry 1000 \
    --retry-delay 1 \
    $ROLLUP_CLIENT_HTTP