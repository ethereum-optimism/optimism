test:
	go test -v ./...

lint:
	golangci-lint run -E asciicheck,goimports,misspell ./...

generate-mocks:
	go generate ./...

.PHONY: \
	test \
	lint \
	generate-mocks
