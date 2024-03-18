GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-bootnode:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-bootnode ./cmd

clean:
	rm -f bin/op-bootnode

test:
	go test -v ./...

.PHONY: \
	op-bootnode \
	clean \
	test
