GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGSSTRING +=-X main.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

# Check if TARGETARCH is set to arm64 and adjust compilers accordingly
ifeq ($(TARGETARCH),arm64)
    CC := aarch64-linux-gnu-gcc
    CXX := aarch64-linux-gnu-g++
endif

op-supervisor:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CC=$(CC) CXX=$(CXX) CGO_ENABLED=0 go build -v $(LDFLAGS) -o ./bin/op-supervisor ./cmd

clean:
	rm bin/op-supervisor

test:
	go test -v ./...

.PHONY: \
	op-supervisor \
	clean \
	test
