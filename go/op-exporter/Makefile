SHELL := /bin/bash

ifndef VERSION
VERSION := `jq .version package.json `
endif
ifndef GITCOMMIT
GITCOMMIT := `git rev-parse HEAD`
endif
ifndef BUILDDATE
BUILDDATE := `date +%Y-%m-%d`
endif

LDFLAGSSTRING :=-X github.com/ethereum-optimism/optimism/go/op_exporter/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/go/op_exporter/version.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/go/op_exporter/version.BuildDate=$(BUILDDATE)

LDFLAGS :=-ldflags "$(LDFLAGSSTRING)"

.PHONY: all build

all: build

# Build binary
build:
	CGO_ENABLED=0 go build $(LDFLAGS)