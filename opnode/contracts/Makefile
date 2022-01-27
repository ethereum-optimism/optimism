SHELL := /bin/bash

CONTRACTS_PATH := "../../packages/contracts/artifacts/contracts"

binding: abi
	$(eval temp := $(shell mktemp))

	cat abis/DepositFeed.json \
		| jq -r .bytecode > $(temp)

	cat abis/DepositFeed.json \
		| jq .abi \
		| abigen --pkg deposit \
		--abi - \
		--out deposit/deposit_feed_raw.go \
		--type deposit \
		--bin $(temp)

	$(eval deployedBytecode := $(shell cat abis/DepositFeed.json | jq -r .deployedBytecode))
	echo "// Code generated - DO NOT EDIT." > deposit/deposit_feed_deployed.go
	echo "// This file is a generated binding and any manual changes will be lost." >> deposit/deposit_feed_deployed.go
	echo "package deposit" >> deposit/deposit_feed_deployed.go
	echo "var DepositDeployedBin = \"$(deployedBytecode)\""  >> deposit/deposit_feed_deployed.go
	gofmt -s -w deposit/deposit_feed_deployed.go

	rm $(temp)

	$(eval temp := $(shell mktemp))

	cat abis/L1Block.json \
		| jq -r .bytecode > $(temp)

	cat abis/L1Block.json \
		| jq .abi \
		| abigen --pkg l1block \
		--abi - \
		--out l1block/l1_block_info_raw.go \
		--type l1block \
		--bin $(temp)

	$(eval deployedBytecode := $(shell cat abis/L1Block.json | jq -r .deployedBytecode))
	echo "// Code generated - DO NOT EDIT." > l1block/l1_block_info_deployed.go
	echo "// This file is a generated binding and any manual changes will be lost." >> l1block/l1_block_info_deployed.go
	echo "package l1block" >> l1block/l1_block_info_deployed.go
	echo "var L1blockDeployedBin = \"$(deployedBytecode)\""  >> l1block/l1_block_info_deployed.go
	gofmt -s -w l1block/l1_block_info_deployed.go

	rm $(temp)

abi:
	cat $(CONTRACTS_PATH)/L1/DepositFeed.sol/DepositFeed.json \
		| jq '{abi,bytecode,deployedBytecode}' \
		> abis/DepositFeed.json
	cat $(CONTRACTS_PATH)/L2/L1Block.sol/L1Block.json \
		| jq '{abi,bytecode,deployedBytecode}' \
		> abis/L1Block.json
