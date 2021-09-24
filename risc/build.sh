#!/bin/bash
cd ../minigeth
GOOS=linux GOARCH=mips go build
cp go-ethereum ../risc/go-ethereum
cd ../risc
file go-ethereum

#GOOS=linux GOARCH=mips go build test.go
