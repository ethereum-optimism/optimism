GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

opnode:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/opnode ./cmd/main.go

clean:
	rm bin/opnode

test:
	go test -v ./...

lint:
	golangci-lint run -E asciicheck,goimports,misspell ./...

.PHONY: \
	bss \
	clean \
	test \
	lint
