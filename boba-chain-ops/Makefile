.PHONY: boba-migrate boba-regenerate boba-rollover boba-crawler boba-devnet build test

test:
	go test ./...

build:
	go build -o ./bin/boba-migrate ./cmd/boba-migrate && \
	go build -o ./bin/boba-regenerate ./cmd/boba-regenerate && \
	go build -o ./bin/boba-rollover ./cmd/boba-rollover && \
	go build -o ./bin/boba-crawler ./cmd/boba-crawler && \
	go build -o ./bin/boba-devnet ./cmd/boba-devnet

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
