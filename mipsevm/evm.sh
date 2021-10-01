#!/bin/bash -e
(cd ../ && npx hardhat compile) && go build && ./mipsevm $1
