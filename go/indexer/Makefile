GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

L1BRIDGE_ABI_ARTIFACT = ../../packages/contracts/artifacts/contracts/L1/messaging/L1StandardBridge.sol/L1StandardBridge.json
L2BRIDGE_ABI_ARTIFACT = ../../packages/contracts/artifacts/contracts/L2/messaging/L2StandardBridge.sol/L2StandardBridge.json

ERC20_ABI_ARTIFACT = ./contracts/ERC20.sol/ERC20.json

SCC_ABI_ARTIFACT = ../../packages/contracts/artifacts/contracts/L1/rollup/StateCommitmentChain.sol/StateCommitmentChain.json

indexer:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/indexer

clean:
	rm indexer

test:
	go test -v ./...

lint:
	golangci-lint run ./...

bindings: bindings-l1bridge bindings-l2bridge bindings-l1erc20 bindings-l2erc20 bindings-scc bindings-address-manager

bindings-l1bridge:
	$(eval temp := $(shell mktemp))

	cat $(L1BRIDGE_ABI_ARTIFACT) \
		| jq -r .bytecode > $(temp)

	cat $(L1BRIDGE_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg l1bridge \
		--abi - \
		--out bindings/l1bridge/l1_standard_bridge.go \
		--type L1StandardBridge \
		--bin $(temp)

	rm $(temp)

bindings-l2bridge:
	$(eval temp := $(shell mktemp))

	cat $(L2BRIDGE_ABI_ARTIFACT) \
		| jq -r .bytecode > $(temp)

	cat $(L2BRIDGE_ABI_ARTIFACT) \
		| jq .abi \
		| ../../l2geth/build/bin/abigen --pkg l2bridge \
		--abi - \
		--out bindings/l2bridge/l2_standard_bridge.go \
		--type L2StandardBridge \
		--bin $(temp)

	rm $(temp)

bindings-l1erc20:
	$(eval temp := $(shell mktemp))

	cat $(ERC20_ABI_ARTIFACT) \
		| jq -r .bytecode > $(temp)

	cat $(ERC20_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg l1erc20 \
		--abi - \
		--out bindings/l1erc20/l1erc20.go \
		--type L1ERC20 \
		--bin $(temp)

	rm $(temp)

bindings-l2erc20:
	$(eval temp := $(shell mktemp))

	cat $(ERC20_ABI_ARTIFACT) \
		| jq -r .bytecode > $(temp)

	cat $(ERC20_ABI_ARTIFACT) \
		| jq .abi \
		| ../../l2geth/build/bin/abigen --pkg l2erc20 \
		--abi - \
		--out bindings/l2erc20/l2erc20.go \
		--type L2ERC20 \
		--bin $(temp)

	rm $(temp)

bindings-scc:
	$(eval temp := $(shell mktemp))

	cat $(SCC_ABI_ARTIFACT) \
		| jq -r .bytecode > $(temp)

	cat $(SCC_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg scc \
		--abi - \
		--out bindings/scc/statecommitmentchain.go \
		--type StateCommitmentChain \
		--bin $(temp)

	rm $(temp)

bindings-address-manager:
	$(eval temp := $(shell mktemp))

	cat $(ADDRESS_MANAGER_ABI_ARTIFACT) \
		| jq -r .bytecode > $(temp)

	cat $(ADDRESS_MANAGER_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg address_manager \
		--abi - \
		--out ./bindings/address_manager/address_manager.go \
		--type AddressManager \
		--bin $(temp)

.PHONY: \
	indexer \
	bindings \
	bindings-l1bridge \
	bindings-l2bridge \
	bindings-l1erc20 \
	bindings-l2erc20 \
	bindings-scc \
	bindings-address-manager
	clean \
	test \
	lint
