#!/bin/bash

# Run this script to serve the latest state dump from
# an http server. This is useful to serve the state dump
# to a local instance of the sequencer/verifier during
# development. The state dump can be found at
# `GET /state-dump.latest.json`

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

PYTHON=${PYTHON:-python}
HOST=${HOST:-0.0.0.0}
PORT=${OMGX_DEPLOYER_PORT:-8079}
DIRECTORY=$DIR/../dist/dumps

if [ $SERVE_ONLY == 1 ]
then
    DIRECTORY=$DIR/../deployment/$IF_SERVE_ONLY_EQ_1_THEN_SERVE
    echo "Serving STATIC addresses.json in $DIRECTORY"
else 
    echo "Serving FRESH addresses.json in $DIRECTORY"
fi

if [ ! command -v $PYTHON&>/dev/null ]; then
    echo "Please install python"
    exit 1
fi

VERSION=$($PYTHON --version 2>&1 \
    | cut -d ' ' -f2 \
    | sed -Ee's#([^/]).([^/]).([^/])#\1#')

echo "Found Python version $VERSION"

cd $DIR
echo "Preparing to serve HTTP on $HOST port $PORT"
$PYTHON http_cors.py $PORT $HOST $DIRECTORY