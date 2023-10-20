SHELL := /usr/bin/env bash

pkg := bindings
monorepo-base := $(shell dirname $(realpath .))
contracts-dir := $(monorepo-base)/packages/contracts-bedrock
ETHERSCAN_APIKEY ?=
ETHERSCAN_OP_APIKEY ?=

all: version mkdir bindings

version:
	forge --version
	abigen --version

compile:
	cd $(contracts-dir) && \
		forge clean && \
		pnpm build

bindgen: compile bindgen-generate-all

bindgen-local: compile bindgen-generate-local

bindgen-remote: bindgen-generate-remote

bindgen-generate-all:
	go run ./bindgen/ \
		generate \
		--metadata-out ./$(pkg) \
		--go-package $(pkg) \
		--monorepo-base $(monorepo-base) \
		--log.level info \
		all \
		--local-contracts ./artifacts.json \
		--source-maps-list MIPS,PreimageOracle \
		--forge-artifacts $(contracts-dir)/forge-artifacts \
		--client etherscan \
		--remote-contracts ./artifacts.json \
		--source-chainid 1 \
		--source-apikey $(ETHERSCAN_APIKEY) \
		--compare-chainid 10 \
		--compare-apikey $(ETHERSCAN_OP_APIKEY) \
		--compare-deployment-bytecode \
		--compare-init-bytecode

bindgen-generate-local:
	go run ./bindgen/ \
		generate \
		--metadata-out ./$(pkg) \
		--go-package $(pkg) \
		--monorepo-base $(monorepo-base) \
		--log.level info \
		local \
		--local-contracts ./artifacts.json \
		--source-maps-list MIPS,PreimageOracle \
		--forge-artifacts $(contracts-dir)/forge-artifacts

bindgen-generate-remote:
	go run ./bindgen/ \
		generate \
		--metadata-out ./$(pkg) \
		--go-package $(pkg) \
		--monorepo-base $(monorepo-base) \
		--log.level info \
		remote \
		--client etherscan \
		--remote-contracts ./artifacts.json \
		--source-chainid 1 \
		--source-apikey $(ETHERSCAN_APIKEY) \
		--compare-chainid 10 \
		--compare-apikey $(ETHERSCAN_OP_APIKEY) \
		--compare-deployment-bytecode \
		--compare-init-bytecode

mkdir:
	mkdir -p $(pkg)

clean-contracts:
	cd $(contracts-dir) && \
		pnpm clean

clean:
	rm -rf $(pkg)

test:
	go test ./...
