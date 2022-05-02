SHELL := /bin/bash

CONTRACTS_PATH := "../../packages/contracts/"
OPTIMISM_PORTAL := ../../packages/contracts/contracts/L1/OptimismPortal.sol
L1_BLOCK_INFO := ../../packages/contracts/contracts/L2/L1Block.sol


bindings: bindings-l1-block-info bindings-optimism-portal deployed-bin-l1-block-info


# Split up b/c I don't know how to include this step in the L1 Block Info Bindings
# What is occuring is that the `temp` variable is hard to pull into the `eval`
deployed-bin-l1-block-info: bindings-l1-block-info
	$(eval deployedBytecode := $(shell cat bin/l1block_deployed.hex))
	echo "// Code generated - DO NOT EDIT." > l1block/l1_block_info_deployed.go
	echo "// This file is a generated binding and any manual changes will be lost." >> l1block/l1_block_info_deployed.go
	echo "package l1block" >> l1block/l1_block_info_deployed.go
	echo "var L1blockDeployedBin = \"$(deployedBytecode)\""  >> l1block/l1_block_info_deployed.go
	gofmt -s -w l1block/l1_block_info_deployed.go

bindings-l1-block-info:
	$(eval temp := $(shell mktemp -d))
	solc \
		--abi $(L1_BLOCK_INFO) \
		--bin $(L1_BLOCK_INFO) \
		--combined-json  abi,bin,bin-runtime \
		--allow-paths $(CONTRACTS_PATH) \
		-o $(temp)
	abigen \
		--combined-json $(temp)/combined.json \
		--pkg l1block \
		--type l1block \
		--out ./l1block/l1_block_info_raw.go
	cat $(temp)/combined.json | jq -r ".contracts | with_entries(select(.key | match(\"L1Block\")))[] | .\"bin-runtime\"" > bin/l1block_deployed.hex


bindings-optimism-portal:
	$(eval temp := $(shell mktemp -d))
	solc \
		--abi $(OPTIMISM_PORTAL) \
		--bin $(OPTIMISM_PORTAL) \
		--combined-json  abi,bin,bin-runtime \
		--allow-paths $(CONTRACTS_PATH) \
		-o $(temp)
	abigen \
		--combined-json $(temp)/combined.json \
		--pkg deposit \
		--type deposit \
		--out ./deposit/deposit_feed_raw.go