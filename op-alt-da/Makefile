GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

da-server:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CGO_ENABLED=0 go build -v $(LDFLAGS) -o ./bin/da-server ./cmd/daserver

clean:
	rm bin/da-server

test:
	go test -v ./...

.PHONY: \
	op-batcher \
	clean \
	test
