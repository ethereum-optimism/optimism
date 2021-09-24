#!/bin/bash
cd ../minigeth
GOOS=linux GOARCH=riscv64 go build
cp go-ethereum ../risc/go-ethereum
cd ../risc
file go-ethereum

