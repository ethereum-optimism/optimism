#!/usr/bin/env bash
set -e

(cd minigeth/ && go build)
mkdir -p /tmp/cannon

# london starts at 12965000
BLOCK=$1

while [ true ]
do
  minigeth/go-ethereum $BLOCK
  ((BLOCK=BLOCK+1))
done
