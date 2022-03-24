#!/usr/bin/env bash
FILES=$(grep -v "#" files_minigeth)
MINIGETH=$PWD/minigeth
git clone https://github.com/ethereum/go-ethereum.git
cd go-ethereum
git checkout 26675454bf93bf904be7a43cce6b3f550115ff90
rsync -Rv $FILES $MINIGETH
