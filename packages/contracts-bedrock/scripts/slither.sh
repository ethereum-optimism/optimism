#!/bin/bash

if [ ! -d forge-artifacts/build-info ]; then
    npx hardhat compile
fi

cp -rf forge-artifacts/build-info artifacts/build-info
slither .
