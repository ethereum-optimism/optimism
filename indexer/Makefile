GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

indexer:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/indexer

indexer-refresh:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/indexer-refresh

clean:
	rm indexer && rm indexer-refresh

test:
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	indexer \
	indexer-refresh \
	bindings \
	bindings-scc \
	clean \
	test \
	lint
