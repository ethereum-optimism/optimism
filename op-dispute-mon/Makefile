GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-dispute-mon/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-dispute-mon/version.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

# Check if TARGETARCH is set to arm64 and adjust compilers accordingly
ifeq ($(TARGETARCH),arm64)
    CC := aarch64-linux-gnu-gcc
    CXX := aarch64-linux-gnu-g++
endif

op-dispute-mon:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CC=$(CC) CXX=$(CXX) go build -v $(LDFLAGS) -o ./bin/op-dispute-mon ./cmd
.PHONY: op-dispute-mon

clean:
	rm bin/op-dispute-mon
.PHONY: clean

test:
	go test -v ./...
.PHONY: test
