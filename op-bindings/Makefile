SHELL := /bin/bash

pkg := bindings
contracts-dir := ../packages/contracts-bedrock

all: version mkdir bindings

version:
	forge --version
	abigen --version

compile:
	cd $(contracts-dir) && \
		yarn build

bindings: compile bindings-build

bindings-build:
	go run ./gen/main.go \
		-forge-artifacts ../packages/contracts-bedrock/forge-artifacts \
		-out ./bindings \
		-contracts ./artifacts.json \
		-source-maps MIPS,PreimageOracle \
		-package $(pkg)

mkdir:
	mkdir -p $(pkg)

clean-contracts:
	cd $(contracts-dir) && \
		yarn clean

clean:
	rm -rf $(pkg)

test:
	go test ./...
