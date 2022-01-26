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

	rm $(temp)

abi:
	cat $(CONTRACTS_PATH)/L1/DepositFeed.sol/DepositFeed.json \
		| jq '{abi,bytecode}' \
		> abis/DepositFeed.json
	cat $(CONTRACTS_PATH)/L2/L1Block.sol/L1Block.json \
		| jq '{abi,bytecode}' \
		> abis/L1Block.json