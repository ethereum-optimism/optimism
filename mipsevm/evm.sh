#!/bin/bash
(cd ../ && npx hardhat compile) && go build && ./mipsevm $1
