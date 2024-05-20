GOTAGS = nosilkworm

test:
	go test -tags $(GOTAGS) ./...
.PHONY: test

build:
	go build -tags $(GOTAGS) -o ./bin/boba-migrate ./cmd/boba-migrate && \
	go build -tags $(GOTAGS) -o ./bin/boba-regenerate ./cmd/boba-regenerate && \
	go build -tags $(GOTAGS) -o ./bin/boba-crawler ./cmd/boba-crawler && \
	go build -tags $(GOTAGS) -o ./bin/boba-devnet ./cmd/boba-devnet && \
	go build -tags $(GOTAGS) -o ./bin/check-l2 ./cmd/check-l2
.PHONY: build

boba-migrate:
	go build -tags $(GOTAGS) -o ./bin/boba-migrate ./cmd/boba-migrate
.PHONY: boba-migrate

boba-regenerate:
	go build -tags $(GOTAGS) -o ./bin/boba-regenerate ./cmd/boba-regenerate
.PHONY: boba-regenerate

boba-crawler:
	go build -tags $(GOTAGS) -o ./bin/boba-crawler ./cmd/boba-crawler
.PHONY: boba-crawler

boba-devnet:
	go build -tags $(GOTAGS) -o ./bin/boba-devnet ./cmd/boba-devnet
.PHONY: boba-devnet

check-l2:
	go build -tags $(GOTAGS) -o ./bin/check-l2 ./cmd/check-l2
.PHONY: check-l2

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./...
.PHONY: lint
