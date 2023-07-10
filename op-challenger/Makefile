GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-challenger:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-challenger ./cmd

clean:
	rm bin/op-challenger

test:
	go test -v ./...

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint -e "errors.As" -e "errors.Is"

devnet-challenger:
	OP_CHALLENGER_MNEUMONIC="test test test test test test test test test test test junk" \
	OP_CHALLENGER_HD_PATH="m/44'/60'/0'/0/3" \
	OP_CHALLENGER_L1_ETH_RPC="http://localhost:8545" \
	OP_CHALLENGER_GAME_ADDRESS="0x0000000" \
	OP_CHALLENGER_ALPHABET="abcdefghijklmnopqrstuvwxyz" \
	go run ./cmd

.PHONY: \
	clean \
	op-challenger \
	test \
	lint
