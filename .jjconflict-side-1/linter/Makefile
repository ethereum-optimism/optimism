BIN := bin/op-golangci-lint
SRC := $(shell find analyzers -type f)

build: $(BIN)

$(BIN): $(SRC) .custom-gcl.yml | bin
	golangci-lint custom -v

bin:
	mkdir -p bin

test:
	go test ./...

.PHONY: build test
