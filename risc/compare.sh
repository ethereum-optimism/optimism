#!/bin/bash
echo "running in go"
$(cd ../mipsevm && ./evm.sh /tmp/minigeth.bin > /tmp/gethtrace)
echo "compare"
./simple.py
