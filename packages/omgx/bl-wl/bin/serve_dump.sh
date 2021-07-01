#!/bin/bash

# Run this script to serve the whitelisting addresses file from
# an http server.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

PYTHON=${PYTHON:-python}
HOST=${HOST:-0.0.0.0}
PORT=${BL_WL_PORT:-8079}
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
