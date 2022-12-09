GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

all: build

docker:
	docker build ../ -f Dockerfile -t signer:latest

build:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/signer ./cmd

clean:
	rm ./bin/signer

test:
	[ '$(shell mockgen --version)' = 'v1.6.0' ] || go install github.com/golang/mock/mockgen@v1.6.0
	go generate ./...
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	build \
	clean \
	test \
	lint \
	docker
