GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

# Check if TARGETARCH is set to arm64 and adjust compilers accordingly
ifeq ($(TARGETARCH),arm64)
    CC := aarch64-linux-gnu-gcc
    CXX := aarch64-linux-gnu-g++
endif

op-heartbeat:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CC=$(CC) CXX=$(CXX) CGO_ENABLED=0 go build -v $(LDFLAGS) -o ./bin/op-heartbeat ./cmd

clean:
	rm bin/op-heartbeat

test:
	go test -v ./...

.PHONY: \
	clean \
	op-heartbeat \
	test
