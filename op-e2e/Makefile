test:
	go test -v ./...

test-erigon:
	go test -v --erigon -timeout 1h

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint -e "errors.As" -e "errors.Is"

.PHONY: \
	test \
	test-erigon \
	lint
