#!/bin/bash -e
cd ../minigeth
export GOOS=linux
export GOARCH=mips
export GOMIPS=softfloat
go build
cp go-ethereum ../risc/go-ethereum
cd ../risc
file go-ethereum

#GOOS=linux GOARCH=mips go build test.go
