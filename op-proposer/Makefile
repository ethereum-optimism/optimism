GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
# Find the latest github tag that contains this commit. If none are found, set the version string to "untagged"
VERSION ?= $(shell tag=$$(git tag --contains $(GITCOMMIT) | grep '^op-proposer/' | sort -V | tail -n 1 | sed 's/op-proposer\///'); \
             if [ -z "$$tag" ]; then \
                 echo "untagged"; \
             else \
                 echo $$tag; \
             fi)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-proposer:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-proposer ./cmd

clean:
	rm bin/op-proposer

test:
	go test -v ./...

.PHONY: \
	clean \
	op-proposer \
	test
