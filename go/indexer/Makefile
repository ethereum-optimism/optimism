GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

L1BRIDGE_ABI_ARTIFACT = ../../packages/contracts/artifacts/contracts/L1/messaging/L1StandardBridge.sol/L1StandardBridge.json
L2BRIDGE_ABI_ARTIFACT = ../../packages/contracts/artifacts/contracts/L2/messaging/L2StandardBridge.sol/L2StandardBridge.json

indexer:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/indexer

clean:
	rm indexer

test:
	go test -v ./...

lint:
	golangci-lint run ./...

bindings: bindings-l1bridge bindings-l2bridge

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


.PHONY: \
	indexer \
	bindings \
	bindings-l1bridge \
	bindings-l2bridge \
	clean \
	test \
	lint
