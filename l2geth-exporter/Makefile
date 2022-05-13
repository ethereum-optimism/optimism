SHELL := /bin/bash

VERSION := `git describe --abbrev=0`
GITCOMMIT := `git rev-parse HEAD`
BUILDDATE := `date +%Y-%m-%d`
BUILDUSER := `whoami`

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS :=-ldflags "$(LDFLAGSSTRING)"

all: build

build:
	CGO_ENABLED=0 go build $(LDFLAGS)

clean:
	rm l2geth-exporter

test:
	go test -v ./...

lint:
	golangci-lint run ./...

binding:
	$(eval tempCTC := $(shell mktemp))

	cat ../../packages/contracts/deployments/mainnet/CanonicalTransactionChain.json \
		| jq -r .bytecode > $(tempCTC)

	cat ../../packages/contracts/deployments/mainnet/CanonicalTransactionChain.json \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/CanonicalTransactionChain.go \
		--type CanonicalTransactionChain \
		--bin $(tempCTC)

	rm $(tempCTC)

	$(eval tempSCC := $(shell mktemp))

	cat ../../packages/contracts/deployments/mainnet/StateCommitmentChain.json \
		| jq -r .bytecode > $(tempSCC)

	cat ../../packages/contracts/deployments/mainnet/StateCommitmentChain.json \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/StateCommitmentChain.go \
		--type StateCommitmentChain \
		--bin $(tempSCC)

	rm $(tempSCC)