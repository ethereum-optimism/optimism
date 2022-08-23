GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

indexer:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/indexer

clean:
	rm indexer

test:
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	indexer \
	clean \
	test \
	lint
