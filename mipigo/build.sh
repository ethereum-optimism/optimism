#!/bin/bash -e
cd ../minigeth
export GOOS=linux
export GOARCH=mips
export GOMIPS=softfloat
go build
cd ../mipigo

cp ../minigeth/go-ethereum minigeth
file minigeth

if [[ ! -d venv ]]; then
    python3 -m venv venv
fi

source venv/bin/activate
pip3 install -r requirements.txt
./compile.py
deactivate
