GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGSSTRING +=-X main.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-supervisor:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CGO_ENABLED=0 go build -v $(LDFLAGS) -o ./bin/op-supervisor ./cmd

clean:
	rm bin/op-supervisor

test:
	go test -v ./...

.PHONY: \
	op-supervisor \
	clean \
	test
