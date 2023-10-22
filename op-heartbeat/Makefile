GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-heartbeat:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/op-heartbeat ./cmd

clean:
	rm bin/op-heartbeat

test:
	go test -v ./...

.PHONY: \
	clean \
	op-heartbeat \
	test \
	lint
