#!/bin/bash -e
(cd ../ && npx hardhat compile)
(cd ../risc && ./build.sh && COMPILE=1 ./run.py)
go build
./mipsevm /tmp/minigeth.bin
