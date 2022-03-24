#!/usr/bin/env bash
set -e

(cd minigeth/ && go build)
mkdir -p /tmp/cannon

# 0 tx:         13284491
# low tx:       13284469
# delete issue: 13284053
if [ $# -eq 0 ]; then
  BLOCK=13284469
else
  BLOCK=$1
fi

minigeth/go-ethereum $BLOCK

