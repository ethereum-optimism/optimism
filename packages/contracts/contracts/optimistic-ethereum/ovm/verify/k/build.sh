#!/bin/bash
CDD=$PWD
cd ../../../
solc --allow-paths . --combined-json=abi,bin,bin-runtime,srcmap,srcmap-runtime,ast ovm/SafetyChecker.sol > $CDD/dapp/out/SafetyChecker.sol.json


