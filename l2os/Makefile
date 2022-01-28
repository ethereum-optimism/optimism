GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

l2os:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/l2os

clean:
	rm l2os

test:
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	clean \
	l2os \
	test \
	lint
