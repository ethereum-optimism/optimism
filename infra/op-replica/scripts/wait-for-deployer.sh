#!/bin/bash
set -eou
if [[ -z $URL ]]; then
    echo "Must pass URL"
    exit 1
fi
echo "Waiting for Deployer"
curl \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry 1000 \
    --retry-delay 1 \
    $URL
echo "Contracts deployed"