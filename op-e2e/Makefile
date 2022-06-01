test:
	go test -v ./...

lint:
	golangci-lint run -E asciicheck,goimports,misspell ./...

.PHONY: \
	test \
	lint
