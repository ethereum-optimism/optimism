#!/bin/bash
echo "compiling"
./build.sh
COMPILE=1 ./run.py
echo "running in go"
$(cd ../mipsevm && DEBUG=1 ./evm.sh /tmp/minigeth.bin > /tmp/gethtrace)
echo "compare"
./simple.py
