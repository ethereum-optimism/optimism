.PHONY: boba-migrate boba-regenerate boba-rollover boba-crawler build test

test:
	go test ./...

build:
	env GO111MODULE=on go build -o ./bin ./cmd/boba-migrate && \
	env GO111MODULE=on go build -o ./bin ./cmd/boba-regenerate && \
	env GO111MODULE=on go build -o ./bin ./cmd/boba-rollover && \
	env GO111MODULE=on go build -o ./bin ./cmd/boba-crawler

boba-migrate:
	env GO111MODULE=on go build -o ./bin ./cmd/boba-migrate

boba-regenerate:
	env GO111MODULE=on go build -o ./bin ./cmd/boba-regenerate

boba-rollover:
	env GO111MODULE=on go build -o ./bin ./cmd/boba-rollover

boba-crawler:
	env GO111MODULE=on go build -o ./bin ./cmd/boba-crawler
