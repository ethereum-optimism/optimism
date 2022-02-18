GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
GITVERSION := $(shell cat package.json | jq .version)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

teleportr:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/teleportr

clean:
	rm teleportr

test:
	go test -v ./...

lint: 
	golangci-lint run ./...

.PHONY: \
	teleportr \
	clean \
	test \
	lint
