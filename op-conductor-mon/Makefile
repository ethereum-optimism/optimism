GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DATE := $(shell git show -s --format='%ct')
OP_CONDUCTOR_MON_VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GIT_COMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDGIT_DATEATE)
LDFLAGSSTRING +=-X main.Version=$(OP_CONDUCTOR_MON_VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

all: build

docker:
	docker build ../ -f Dockerfile -t op-conductor-mon:latest

build:
	env GO111MODULE=on go build -v $(LDFLAGS) -o ./bin/op-conductor-mon ./cmd/monitor

clean:
	rm ./bin/op-conductor-mon

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
