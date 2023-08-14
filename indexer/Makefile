GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
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
	bindings \
	bindings-scc \
	clean \
	test \
	lint
