GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

batch-submitter:
	env GO111MODULE=on go build $(LDFLAGS) ./cmd/batch-submitter

clean:
	rm batch-submitter

test:
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	batch-submitter \
	clean \
	test \
	lint
