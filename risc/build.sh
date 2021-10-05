#!/bin/bash -e
cd ../minigeth
export GOOS=linux
export GOARCH=mips
export GOMIPS=softfloat

# brew install llvm
#export CGO_ENABLED=1
#export CC="$PWD/../risc/clangwrap.sh"

go build -v
cp go-ethereum ../risc/go-ethereum
cd ../risc
file go-ethereum

#GOOS=linux GOARCH=mips go build test.go
