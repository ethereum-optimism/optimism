GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

CTC_ABI_ARTIFACT := ../../packages/contracts/artifacts/contracts/L1/rollup/CanonicalTransactionChain.sol/CanonicalTransactionChain.json
SCC_ABI_ARTIFACT := ../../packages/contracts/artifacts/contracts/L1/rollup/StateCommitmentChain.sol/StateCommitmentChain.json
L2BRIDGE_ABI_ARTIFACT = ../../packages/contracts/artifacts/contracts/L2/messaging/L2StandardBridge.sol/L2StandardBridge.json

indexer:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/indexer

clean:
	rm indexer

test:
	go test -v ./...

lint:
	golangci-lint run ./...

bindings: bindings-ctc bindings-scc bindings-l2bridge

bindings-ctc:
	$(eval temp := $(shell mktemp))

	cat $(CTC_ABI_ARTIFACT) \
		| jq -r .bytecode > $(temp)

	cat $(CTC_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg ctc \
		--abi - \
		--out bindings/ctc/canonical_transaction_chain.go \
		--type CanonicalTransactionChain \
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
		--out bindings/scc/state_commitment_chain.go \
		--type StateCommitmentChain \
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
	bindings-ctc \
	bindings-scc \
	bindings-l2bridge \
	clean \
	test \
	lint
