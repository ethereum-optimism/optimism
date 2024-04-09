GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
# Find the github tag that points to this commit. If none are found, set the version string to "untagged"
VERSION ?= $(shell tag=$$(git tag --points-at $(GITCOMMIT) | grep '^op-batcher/' | sort -V | tail -n 1 | sed 's/op-batcher\///'); \
             if [ -z "$$tag" ]; then \
                 echo "untagged"; \
             else \
                 echo $$tag; \
             fi)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-batcher:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-batcher ./cmd

clean:
	rm bin/op-batcher

test:
	go test -v ./...

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzChannelConfig_CheckTimeout ./batcher
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDurationZero ./batcher
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDurationTimeoutMaxChannelDuration ./batcher
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDurationTimeoutZeroMaxChannelDuration ./batcher
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzChannelCloseTimeout ./batcher
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzChannelZeroCloseTimeout ./batcher
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzSeqWindowClose ./batcher
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzSeqWindowZeroTimeoutClose ./batcher

.PHONY: \
	op-batcher \
	clean \
	test \
	fuzz
