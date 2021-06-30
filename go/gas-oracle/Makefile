SHELL := /bin/bash

GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS :=-ldflags "$(LDFLAGSSTRING)"

gas-oracle:
	env GO111MODULE=on go build $(LDFLAGS)

clean:
	rm gas-oracle

test:
	go test -v ./...

lint:
	golangci-lint run ./...

binding:
	$(eval temp := $(shell mktemp))

	cat abis/OVM_GasPriceOracle.json \
		| jq -r .bin > $(temp)

	cat abis/OVM_GasPriceOracle.json \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/gaspriceoracle.go \
		--type GasPriceOracle \
		--bin $(temp)

	rm $(temp)
