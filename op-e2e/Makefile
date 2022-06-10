test:
	go test -v ./...

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint,exhaustive ./...

.PHONY: \
	test \
	lint
