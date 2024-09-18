#!/bin/sh

set -euo

forge build

cd build/emit.sol
cat EmitEvent.json | jq -r '.bytecode.object' > EmitEvent.bin
cat EmitEvent.json | jq '.abi' > EmitEvent.abi
cd ../..

abigen --abi ./build/emit.sol/EmitEvent.abi --bin ./build/emit.sol/EmitEvent.bin --pkg emit --out ./emit.go
