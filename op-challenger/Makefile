GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-challenger:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-challenger ./cmd
.PHONY: op-challenger

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzKeccak ./game/keccak/matrix
.PHONY: fuzz

clean:
	rm bin/op-challenger
.PHONY: clean

hex2bytes-check:
	@grep -Frq "common.Hex2Bytes" . && echo "common.Hex2Bytes found in op-challenger files" && exit 1 || echo "common.Hex2Bytes not found"
.PHONY: hex2bytes-check

test:
	go test -v ./...
.PHONY: test

visualize:
	./scripts/visualize.sh
.PHONY: visualize
