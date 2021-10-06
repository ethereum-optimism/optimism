#!/bin/bash -e
(cd ../ && npx hardhat compile)
(cd ../mipigeth && ./build.sh)
go build
./mipsevm ../mipigeth/minigeth.bin
