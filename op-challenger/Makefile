.PHONY: clean tidy format lint build run test

all: clean tidy format golangci lint build start-devnet run

ci: clean tidy format golangci lint build test

start-devnet:
	./docker/start_devnet.sh

clean:
	rm -rf bin/op-challenger
	go clean -cache
	go clean -modcache

golangci:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

format:
	gofmt -s -w -l .

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint -e "errors.As" -e "errors.Is" --timeout 5m

build:
	env GO111MODULE=on go build -o bin/op-challenger ./cmd

run:
	make build
	bin/op-challenger

test:
	go test -v ./...

tidy:
	go mod tidy
