#!/bin/sh

set -euo

forge build

cd build/Invoker.sol
cat Invoker.json | jq -r '.bytecode.object' > Invoker.bin
cat Invoker.json | jq '.abi' > Invoker.abi
cd ../../

mkdir -p bindings/invoker
abigen --abi ./build/Invoker.sol/Invoker.abi --bin ./build/Invoker.sol/Invoker.bin --pkg invoker --out ./bindings/invoker/invoker.go

cd build/DelegateCallProxy.sol
cat DelegateCallProxy.json | jq -r '.bytecode.object' > DelegateCallProxy.bin
cat DelegateCallProxy.json | jq '.abi' > DelegateCallProxy.abi
cd ../../
mkdir -p bindings/delegatecallproxy
abigen --abi ./build/DelegateCallProxy.sol/DelegateCallProxy.abi --bin ./build/DelegateCallProxy.sol/DelegateCallProxy.bin --pkg delegatecallproxy --out ./bindings/delegatecallproxy/delegatecallproxy.go
