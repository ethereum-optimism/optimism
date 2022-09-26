#!/bin/bash

HASH=$1

if [[ -z $HASH ]]; then
    exit 1
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

TRACES=$DIR/call-traces
RECEIPTS=$DIR/receipts
DIFFS=$DIR/state-diffs

mkdir -p $TRACES
mkdir -p $RECEIPTS
mkdir -p $DIFFS

cast rpc \
    debug_traceTransaction \
    $HASH \
    '{"tracer": "callTracer"}' | jq > $TRACES/$HASH.json

cast receipt $HASH --json | jq > $RECEIPTS/$HASH.json

cast rpc \
    debug_traceTransaction \
    $HASH \
    '{"tracer": "prestateTracer"}' | jq > $DIFFS/$HASH.json

