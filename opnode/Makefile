GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimistic-specs/opnode/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimistic-specs/opnode/version.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

opnode:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/opnode ./cmd/main.go

clean:
	rm bin/opnode

test:
	go test -v ./...

lint:
	golangci-lint run -E asciicheck,goimports,misspell ./...

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadUnmarshal ./l2
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadMarshalUnmarshal ./l2
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzL1InfoRoundTrip ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzL1InfoAgainstContract ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzUnmarshallLogEvent ./rollup/derive


.PHONY: \
	bss \
	clean \
	test \
	lint \
	fuzz
