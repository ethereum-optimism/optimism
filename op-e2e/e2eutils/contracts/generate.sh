#!/bin/sh

set -euo

forge build

cd build/emit.sol
cat EmitEvent.json | jq -r '.bytecode.object' > EmitEvent.bin
cat EmitEvent.json | jq '.abi' > EmitEvent.abi
cd ../..

mkdir -p bindings/emit
abigen --abi ./build/emit.sol/EmitEvent.abi --bin ./build/emit.sol/EmitEvent.bin --pkg emit --out ./bindings/emit/emit.go

cd build/ICrossL2Inbox.sol
cat ICrossL2Inbox.json | jq -r '.bytecode.object' > ICrossL2Inbox.bin
cat ICrossL2Inbox.json | jq '.abi' > ICrossL2Inbox.abi
cd ../..

mkdir -p bindings/inbox
abigen --abi ./build/ICrossL2Inbox.sol/ICrossL2Inbox.abi --bin ./build/ICrossL2Inbox.sol/ICrossL2Inbox.bin --pkg inbox --out ./bindings/inbox/inbox.go

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
