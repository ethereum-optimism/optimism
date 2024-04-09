GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
# Find the latest github tag that contains this commit. If none are found, set the version string to "untagged"
VERSION ?= $(shell tag=$$(git tag --contains $(GITCOMMIT) | grep '^op-node/' | sort -V | tail -n 1 | sed 's/op-node\///'); \
             if [ -z "$$tag" ]; then \
                 echo "untagged"; \
             else \
                 echo $$tag; \
             fi)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-node/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-node/version.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-node:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-node ./cmd/main.go

clean:
	rm bin/op-node

test:
	go test -v ./...

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzL1InfoBedrockRoundTrip ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzL1InfoEcotoneRoundTrip ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzL1InfoAgainstContract ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzUnmarshallLogEvent ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzParseFrames ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzFrameUnmarshalBinary ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzBatchRoundTrip ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDeriveDepositsRoundTrip ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDeriveDepositsBadVersion ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzParseL1InfoDepositTxDataValid ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzParseL1InfoDepositTxDataBadLength ./rollup/derive
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzRejectCreateBlockBadTimestamp ./rollup/driver
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDecodeDepositTxDataToL1Info ./rollup/driver

generate-mocks:
	go generate ./...

.PHONY: \
	op-node \
	clean \
	test \
	fuzz
