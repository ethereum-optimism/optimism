SHELL := /bin/bash
CONTRACTS_PATH := "../../packages/contracts/"


all: bindings deployed

bindings: bindings-l1-block-info bindings-optimism-portal bindings-L2ToL1MessagePasser
deployed: deployed-bin-l1-block-info deployed-bin-optimism-portal deployed-bin-L2ToL1MessagePasser

# Deployed bindings are split because they depend on the bin file
# TODO: Can probably do this with recipes
deployed-bin-l1-block-info: bindings-l1-block-info
	$(eval deployedBytecode := $(shell cat bin/l1block_deployed.hex))
	echo "// Code generated - DO NOT EDIT." > l1block/l1_block_info_deployed.go
	echo "// This file is a generated binding and any manual changes will be lost." >> l1block/l1_block_info_deployed.go
	echo "package l1block" >> l1block/l1_block_info_deployed.go
	echo "var L1blockDeployedBin = \"$(deployedBytecode)\""  >> l1block/l1_block_info_deployed.go
	gofmt -s -w l1block/l1_block_info_deployed.go

deployed-bin-optimism-portal: bindings-optimism-portal
	$(eval deployedBytecode := $(shell cat bin/portal_deployed.hex))
	echo "// Code generated - DO NOT EDIT." > deposit/optimism_portal_deployed.go
	echo "// This file is a generated binding and any manual changes will be lost." >> deposit/optimism_portal_deployed.go
	echo "package deposit" >> deposit/optimism_portal_deployed.go
	echo "var OptimismPortalDeployedBin = \"$(deployedBytecode)\""  >> deposit/optimism_portal_deployed.go
	gofmt -s -w deposit/optimism_portal_deployed.go

deployed-bin-L2ToL1MessagePasser: bindings-L2ToL1MessagePasser
	$(eval deployedBytecode := $(shell cat bin/l2_to_l1_message_passer.hex))
	echo "// Code generated - DO NOT EDIT." > withdrawer/withdrawer_deployed.go
	echo "// This file is a generated binding and any manual changes will be lost." >> withdrawer/withdrawer_deployed.go
	echo "package withdrawer" >> withdrawer/withdrawer_deployed.go
	echo "var WithdrawerDeployedBin = \"$(deployedBytecode)\""  >> withdrawer/withdrawer_deployed.go
	gofmt -s -w withdrawer/withdrawer_deployed.go

bindings-l1-block-info:
	$(eval temp := $(shell mktemp -d))
	$(eval cwd := $(shell pwd))
	mkdir -p bin
	cd $(CONTRACTS_PATH); \
		forge build \
		&& forge inspect L1Block abi > $(temp)/L1Block.abi \
		&& forge inspect L1Block bytecode > $(temp)/L1Block.bin \
		&& forge inspect L1Block deployedBytecode > $(cwd)/bin/l1block_deployed.hex
	abigen \
		--abi $(temp)/L1Block.abi \
		--bin $(temp)/L1Block.bin \
		--pkg l1block \
		--type L1Block \
		--out ./l1block/l1_block_info_raw.go

bindings-L2ToL1MessagePasser:
	$(eval temp := $(shell mktemp -d))
	$(eval cwd := $(shell pwd))
	mkdir -p bin
	cd $(CONTRACTS_PATH); \
		forge build \
		&& forge inspect L2ToL1MessagePasser abi > $(temp)/Withdrawer.abi \
		&& forge inspect L2ToL1MessagePasser bytecode > $(temp)/Withdrawer.bin \
		&& forge inspect L2ToL1MessagePasser deployedBytecode > $(cwd)/bin/l2_to_l1_message_passer.hex
	abigen \
		--abi $(temp)/Withdrawer.abi \
		--bin $(temp)/Withdrawer.bin \
		--pkg withdrawer \
		--type Withdrawer \
		--out ./withdrawer/withdrawer_raw.go



bindings-optimism-portal:
	$(eval temp := $(shell mktemp -d))
	$(eval cwd := $(shell pwd))
	mkdir -p bin
	cd $(CONTRACTS_PATH); \
		forge build \
		&& forge inspect OptimismPortal abi > $(temp)/OptimismPortal.abi \
		&& forge inspect OptimismPortal bytecode > $(temp)/OptimismPortal.bin \
		&& forge inspect OptimismPortal deployedBytecode > $(cwd)/bin/portal_deployed.hex
	abigen \
		--abi $(temp)/OptimismPortal.abi \
		--bin $(temp)/OptimismPortal.bin \
		--pkg deposit \
		--type OptimismPortal \
		--out ./deposit/deposit_feed_raw.go

