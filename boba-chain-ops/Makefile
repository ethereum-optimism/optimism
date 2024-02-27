test:
	go test -tags nosilkworm ./...
.PHONY: test

build:
	go build -tags nosilkworm -o ./bin/boba-migrate ./cmd/boba-migrate && \
	go build -tags nosilkworm -o ./bin/boba-regenerate ./cmd/boba-regenerate && \
	go build -tags nosilkworm -o ./bin/boba-crawler ./cmd/boba-crawler && \
	go build -tags nosilkworm -o ./bin/boba-devnet ./cmd/boba-devnet && \
	go build -tags nosilkworm -o ./bin/check-l2 ./cmd/check-l2
.PHONY: build

boba-migrate:
	go build -tags nosilkworm -o ./bin/boba-migrate ./cmd/boba-migrate
.PHONY: boba-migrate

boba-regenerate:
	go build -tags nosilkworm -o ./bin/boba-regenerate ./cmd/boba-regenerate
.PHONY: boba-regenerate

boba-crawler:
	go build -tags nosilkworm -o ./bin/boba-crawler ./cmd/boba-crawler
.PHONY: boba-crawler

boba-devnet:
	go build -tags nosilkworm -o ./bin/boba-devnet ./cmd/boba-devnet
.PHONY: boba-devnet

check-l2:
	go build -tags nosilkworm -o ./bin/check-l2 ./cmd/check-l2
.PHONY: check-l2

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./...
.PHONY: lint
