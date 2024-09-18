#!/bin/sh

set -euo

forge build

cd contracts/emit.sol
cat EmitEvent.json | jq -r '.bytecode.object' > EmitEvent.bin
cat EmitEvent.json | jq '.abi' > EmitEvent.abi
cd ../..

abigen --abi ./test-artifacts/emit.sol/EmitEvent.abi --bin ./test-artifacts/emit.sol/EmitEvent.bin --pkg emit --out ./emit.go
