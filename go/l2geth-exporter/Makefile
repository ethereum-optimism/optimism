SHELL := /bin/bash

GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS :=-ldflags "$(LDFLAGSSTRING)"

l2geth-exporter:
	env GO111MODULE=on go build $(LDFLAGS)

clean:
	rm l2geth-exporter

test:
	go test -v ./...

lint:
	golangci-lint run ./...

binding:
	$(eval temp := $(shell mktemp))

	cat ../../packages/contracts/deployments/mainnet/OVM_CanonicalTransactionChain.json \
		| jq -r .bytecode > $(temp)

	cat ../../packages/contracts/deployments/mainnet/OVM_CanonicalTransactionChain.json \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/OVM_CanonicalTransactionChain.go \
		--type OVMCanonicalTransactionChain \
		--bin $(temp)

	rm $(temp)
