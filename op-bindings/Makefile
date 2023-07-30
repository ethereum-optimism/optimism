SHELL := /bin/bash

pkg := bindings
monorepo-base := $(shell dirname $(realpath .))
contracts-dir := $(monorepo-base)/packages/contracts-bedrock

all: version mkdir bindings

version:
	forge --version
	abigen --version

compile:
	cd $(contracts-dir) && \
		pnpm build

bindings: compile bindings-build

bindings-build:
	go run ./gen/main.go \
		-forge-artifacts $(contracts-dir)/forge-artifacts \
		-out ./bindings \
		-contracts ./artifacts.json \
		-source-maps MIPS,PreimageOracle \
		-package $(pkg) \
		-monorepo-base $(monorepo-base)

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./...

mkdir:
	mkdir -p $(pkg)

clean-contracts:
	cd $(contracts-dir) && \
		pnpm clean

clean:
	rm -rf $(pkg)

test:
	go test ./...
