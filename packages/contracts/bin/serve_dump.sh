#!/bin/bash

# Run this script to serve the latest state dump from
# an http server. This is useful to serve the state dump
# to a local instance of the sequencer/verifier during
# development. The state dump can be found at
# `GET /state-dump.latest.json`

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

PYTHON=${PYTHON:-python}
HOST=${HOST:-0.0.0.0}
PORT=${PORT:-8081}
DIRECTORY=$DIR/../dist/dumps

if [ ! command -v $PYTHON&>/dev/null ]; then
    echo "Please install python"
    exit 1
fi

VERSION=$($PYTHON --version 2>&1 \
    | cut -d ' ' -f2 \
    |  sed -Ee's#([^/]).([^/]).([^/])#\1#')


if [[ $VERSION == 3 ]]; then
    $PYTHON -m http.server \
        --bind $HOST $PORT \
        --directory $DIRECTORY
else
    (
        echo "Serving HTTP on $HOST port $PORT"
        cd $DIRECTORY
        $PYTHON -c \
            'import BaseHTTPServer as bhs, SimpleHTTPServer as shs; bhs.HTTPServer(("'$HOST'", '"$PORT"'), shs.SimpleHTTPRequestHandler).serve_forever()'
    )
fi
