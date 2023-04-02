GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-program:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-program ./cmd/main.go

clean:
	rm -rf bin

test:
	go test -v ./...

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint -e "errors.As" -e "errors.Is"

.PHONY: \
	op-program \
	clean \
	test \
	lint
