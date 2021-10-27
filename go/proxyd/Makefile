GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

proxyd:
	go build -v $(LDFLAGS) -o ./bin/proxyd ./cmd/proxyd
.PHONY: proxyd

fmt:
	go mod tidy
	gofmt -w .
.PHONY: fmt
