LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

ufm:
	go build -v $(LDFLAGS) -o ./bin/ufm ./cmd/ufm
.PHONY: ufm

fmt:
	go mod tidy
	gofmt -w .
.PHONY: fmt

test:
	go test -race -v ./...
.PHONY: test

lint:
	go vet ./...
.PHONY: test
