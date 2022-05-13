GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-proposer:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/op-proposer ./cmd

clean:
	rm bin/op-proposer

test:
	go test -v ./...

lint:
	golangci-lint run -E asciicheck,goimports,misspell ./...

bindings: bindings-l2-output-oracle

bindings-l2-output-oracle:
	$(eval temp := $(shell mktemp -d))
	cd ../packages/contracts; \
		forge build \
		&& forge inspect L2OutputOracle abi > $(temp)/L2OutputOracle.abi \
		&& forge inspect L2OutputOracle bytecode > $(temp)/L2OutputOracle.bin
	abigen \
		--abi $(temp)/L2OutputOracle.abi \
		--bin $(temp)/L2OutputOracle.bin \
		--pkg l2oo \
		--type L2OutputOracle \
		--out ./bindings/l2oo/l2_output_oracle.go

.PHONY: \
	bindings \
	bindings-mock-l2-output-oracle \
	clean \
	op-proposer \
	test \
	lint
