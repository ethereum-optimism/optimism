#!/bin/bash -e
cd ../minigeth
export GOOS=linux
export GOARCH=mips
export GOMIPS=softfloat

# brew install llvm
#export CGO_ENABLED=1
#export CC="/usr/local/opt/llvm/bin/clang -target mips-linux-gnu --sysroot /Users/kafka/build/cannon/risc/sysroot/"
#export CC="/Users/kafka/fun/mips/mips-gcc-4.8.1/bin/mips-elf-gcc"

go build
cp go-ethereum ../risc/go-ethereum
cd ../risc
file go-ethereum

#GOOS=linux GOARCH=mips go build test.go
