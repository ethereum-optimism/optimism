GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-node/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-node/version.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-node:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/op-node ./cmd/main.go

clean:
	rm bin/op-node

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
	op-node \
	clean \
	test \
	lint \
	fuzz
