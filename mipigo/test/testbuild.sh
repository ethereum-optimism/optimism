#!/usr/bin/env bash
set -e

export GOOS=linux
export GOARCH=mips
export GOMIPS=softfloat
go build test.go
../compile.py test
