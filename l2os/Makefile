GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

MOCK_L2_OUTPUT_ORACLE_SOL := ../packages/contracts/contracts/L1/MockL2OutputOracle.sol

l2os:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/l2os

clean:
	rm l2os

test:
	go test -v ./...

lint:
	golangci-lint run ./...

bindings: bindings-mock-l2-output-oracle

bindings-mock-l2-output-oracle:
	$(eval temp := $(shell mktemp -d))
	solc \
		--abi $(MOCK_L2_OUTPUT_ORACLE_SOL) \
		--bin $(MOCK_L2_OUTPUT_ORACLE_SOL) \
		-o $(temp)
	abigen \
		--abi $(temp)/MockL2OutputOracle.abi \
		--bin $(temp)/MockL2OutputOracle.bin \
		--pkg l2oo \
		--type MockL2OutputOracle \
		--out ./bindings/l2oo/mock_l2_output_oracle.go


.PHONY: \
	bindings \
	bindings-mock-l2-output-oracle \
	clean \
	l2os \
	test \
	lint
