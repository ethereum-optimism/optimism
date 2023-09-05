.PHONY: boba-migrate boba-regenerate boba-rollover boba-crawler boba-devnet boba-connect check-l2 build test

test:
	go test ./...

build:
	go build -o ./bin/boba-migrate ./cmd/boba-migrate && \
	go build -o ./bin/boba-regenerate ./cmd/boba-regenerate && \
	go build -o ./bin/boba-rollover ./cmd/boba-rollover && \
	go build -o ./bin/boba-crawler ./cmd/boba-crawler && \
	go build -o ./bin/boba-devnet ./cmd/boba-devnet && \
	go build -o ./bin/boba-connect ./cmd/boba-connect && \
	go build -o ./bin/check-l2 ./cmd/check-l2

boba-migrate:
	go build -o ./bin/boba-migrate ./cmd/boba-migrate

boba-regenerate:
	go build -o ./bin/boba-regenerate ./cmd/boba-regenerate

boba-rollover:
	go build -o ./bin/boba-rollover ./cmd/boba-rollover

boba-crawler:
	go build -o ./bin/boba-crawler ./cmd/boba-crawler

boba-devnet:
	go build -o ./bin/boba-devnet ./cmd/boba-devnet

boba-connect:
	go build -o ./bin/boba-connect ./cmd/boba-connect

check-l2:
	go build -o ./bin/check-l2 ./cmd/check-l2

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./...
