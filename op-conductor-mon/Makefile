GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

all: build

docker:
	docker build ../ -f Dockerfile -t op-conductor-mon:latest

build:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/opcm ./cmd/opcm

clean:
	rm ./bin/opcm

generate:
	[ '$(shell mockgen --version)' = 'v1.6.0' ] || go install github.com/golang/mock/mockgen@v1.6.0
	go generate ./...

test: generate
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	build \
	clean \
	test \
	generate \
	lint \
	docker
