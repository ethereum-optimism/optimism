#!/bin/sh

set -euo

forge build

pushd test-artifacts/emit.sol
cat EmitEvent.json | jq -r '.bytecode.object' > EmitEvent.bin
cat EmitEvent.json | jq '.abi' > EmitEvent.abi
popd

abigen --abi ./test-artifacts/emit.sol/EmitEvent.abi --bin ./test-artifacts/emit.sol/EmitEvent.bin --pkg emit --out ./emit.go
