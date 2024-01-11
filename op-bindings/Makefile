SHELL := /usr/bin/env bash

pkg := bindings
monorepo-base := $(shell dirname $(realpath .))
contracts-dir := $(monorepo-base)/packages/contracts-bedrock
contracts-list := ./artifacts.json
log-level := info
ETHERSCAN_APIKEY_ETH ?=
ETHERSCAN_APIKEY_OP ?=
RPC_URL_ETH ?=
RPC_URL_OP ?=

all: version mkdir bindings

version:
	forge --version
	abigen --version

compile:
	cd $(contracts-dir) && \
		forge clean && \
		pnpm build

bindings: bindgen-local

bindings-build: bindgen-generate-local

bindgen: compile bindgen-generate-all

bindgen-generate-all:
	go run ./cmd/ \
		generate \
		--metadata-out ./$(pkg) \
		--bindings-package $(pkg) \
		--contracts-list $(contracts-list) \
		--log.level $(log-level) \
		all \
		--source-maps-list MIPS,PreimageOracle \
		--forge-artifacts $(contracts-dir)/forge-artifacts \
		--etherscan.apikey.eth $(ETHERSCAN_APIKEY_ETH) \
		--etherscan.apikey.op $(ETHERSCAN_APIKEY_OP) \
		--rpc.url.eth $(RPC_URL_ETH) \
		--rpc.url.op $(RPC_URL_OP)

bindgen-local: compile bindgen-generate-local

bindgen-generate-local:
	go run ./cmd/ \
		generate \
		--metadata-out ./$(pkg) \
		--bindings-package $(pkg) \
		--contracts-list $(contracts-list) \
		--log.level $(log-level) \
		local \
		--source-maps-list MIPS,PreimageOracle \
		--forge-artifacts $(contracts-dir)/forge-artifacts

bindgen-remote:
	go run ./cmd/ \
		generate \
		--metadata-out ./$(pkg) \
		--bindings-package $(pkg) \
		--contracts-list $(contracts-list) \
		--log.level $(log-level) \
		remote \
		--etherscan.apikey.eth $(ETHERSCAN_APIKEY_ETH) \
		--etherscan.apikey.op $(ETHERSCAN_APIKEY_OP) \
		--rpc.url.eth $(RPC_URL_ETH) \
		--rpc.url.op $(RPC_URL_OP)

mkdir:
	mkdir -p $(pkg)

clean-contracts:
	cd $(contracts-dir) && \
		pnpm clean

clean:
	rm -rf $(pkg)

test:
	go test ./...
