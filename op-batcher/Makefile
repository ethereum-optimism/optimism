GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION := v0.0.0

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
