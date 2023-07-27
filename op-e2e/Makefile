test:
	go test -v ./...

test-erigon:
	go test -v --erigon -timeout 10m

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint -e "errors.As" -e "errors.Is"

test-external-%: external_%/shim
	go test -v --externalL2 ./external_$*/shim

external_%/shim: external_%/*.go
	go build -o external_$*/shim ./external_$*/

.PHONY: \
	test \
	test-erigon \
	lint
