#!/bin/bash -e
echo "compiling"
./build.sh
echo "running in go"
export STEPS=100000
$(cd ../mipsevm && DEBUG=1 ./evm.sh ../mipigeth/minigeth.bin > /tmp/gethtrace)
echo "compare"
./simple.py
