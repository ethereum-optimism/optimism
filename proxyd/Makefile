LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

proxyd:
	go build -v $(LDFLAGS) -o ./bin/proxyd ./cmd/proxyd
.PHONY: proxyd

fmt:
	go mod tidy
	gofmt -w .
.PHONY: fmt

test:
	go test -v ./...
.PHONY: test

lint:
	go vet ./...
.PHONY: test

test-fallback:
	go test -v ./... -test.run ^TestFallback$
.PHONY: test-fallback
