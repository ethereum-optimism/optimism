GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION := v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

op-challenger:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/op-challenger ./cmd

clean:
	rm bin/op-challenger

test:
	go test -v ./...

visualize:
	./scripts/visualize.sh

alphabet:
	./scripts/alphabet/init_game.sh
	make alphabet-actors

alphabet-actors:
	@./scripts/parallel.sh \
	./scripts/alphabet/mallory.sh 'mallory' \
	./scripts/alphabet/charlie.sh 'charlie'

.PHONY: \
	clean \
	op-challenger \
	test \
	lint \
	visualize \
	alphabet \
	alphabet-actors
