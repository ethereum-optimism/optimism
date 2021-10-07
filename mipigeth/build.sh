#!/bin/bash -e
cd ../minigeth
export GOOS=linux
export GOARCH=mips
export GOMIPS=softfloat
go build
cd ../mipigeth

cp ../minigeth/go-ethereum minigeth
file minigeth

./compile.py
