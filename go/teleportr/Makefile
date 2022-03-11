GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

DEPOSIT_ARTIFACT := ../../packages/contracts/artifacts/contracts/L1/teleportr/TeleportrDeposit.sol/TeleportrDeposit.json
DISBURSER_ARTIFACT := ../../packages/contracts/artifacts/contracts/L2/teleportr/TeleportrDisburser.sol/TeleportrDisburser.json

teleportr:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/teleportr

teleportr-api:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/teleportr-api

clean:
	rm teleportr
	rm api

test:
	go test -v ./...

lint: 
	golangci-lint run ./...

bindings: bindings-deposit bindings-disburser

bindings-deposit:
	$(eval temp := $(shell mktemp))
	cat $(DEPOSIT_ARTIFACT) | jq -r .bytecode > $(temp)
	cat $(DEPOSIT_ARTIFACT) | jq .abi | \
		abigen \
		--pkg deposit \
		--abi - \
		--out bindings/deposit/teleportr_deposit.go \
		--type TeleportrDeposit \
		--bin $(temp)

bindings-disburser:
	$(eval temp := $(shell mktemp))
	cat $(DISBURSER_ARTIFACT) | jq -r .bytecode > $(temp)
	cat $(DISBURSER_ARTIFACT) | jq .abi | \
		abigen \
		--pkg disburse \
		--abi - \
		--out bindings/disburse/teleportr_disburser.go \
		--type TeleportrDisburser \
		--bin $(temp)

.PHONY: \
	teleportr \
	teleportr-api \
	bindings \
	bindings-deposit \
	bindings-disburser \
	clean \
	test \
	lint
