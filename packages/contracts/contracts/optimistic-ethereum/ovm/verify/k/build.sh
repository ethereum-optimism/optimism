#!/bin/bash
cd dapp
solc --allow-paths . --combined-json=abi,bin,bin-runtime,srcmap,srcmap-runtime,ast src/SafetyChecker.sol > out/dapp.sol.json


