SHELL := /bin/bash

build: contracts
.PHONY: build

contracts:
	cd contracts && forge build
.PHONY: contracts

test:
	cd mipsevm && go test -v ./...
.PHONY: test
