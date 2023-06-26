SHELL := /bin/bash

pkg := bindings

all: version mkdir bindings more

version:
	forge --version
	abigen --version

compile:
	cd ../packages/contracts-bedrock/ && \
		yarn clean && \
		yarn build

bindings: compile
	go run ./gen/main.go \
		-forge-artifacts ../packages/contracts-bedrock/forge-artifacts \
		-out ./bindings \
		-contracts ./artifacts.json \
		-source-maps MIPS,PreimageOracle \
		-package bindings

mkdir:
	mkdir -p bin $(pkg)

clean:
	rm -rf bin $(pkg)

test:
	go test ./...
