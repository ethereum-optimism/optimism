#!/bin/bash -e
cd ../minigeth
export GOOS=linux
export GOARCH=mips
export GOMIPS=softfloat

# brew install llvm
# ...and struggle to build musl
# for 0 gain
#export CGO_ENABLED=1
#export CC="$PWD/../risc/clangwrap.sh"
#go build -v -ldflags "-linkmode external -extldflags -static"

go build
cp go-ethereum ../risc/go-ethereum
cd ../risc
file go-ethereum

# optional (doesn't work because of replacements)
#/usr/local/opt/llvm/bin/llvm-strip go-ethereum 
#file go-ethereum

#GOOS=linux GOARCH=mips go build test.go
