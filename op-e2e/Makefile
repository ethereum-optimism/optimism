devnet-allocs:
	PYTHONPATH=../bedrock-devnet python3 ../bedrock-devnet/main.py --monorepo-dir=.. --allocs

pre-test:
	$(shell ../ops/scripts/newer-file.sh ../.devnet/allocs-l1.json .././packages/contracts-bedrock)
	if [ $(.SHELLSTATUS) -ne 0 ]; then \
		make devnet-allocs; \
	fi

test: pre-test
	go test -v ./...

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint -e "errors.As" -e "errors.Is"

test-external-%: pre-test external_%/shim
	go test -v --externalL2 ./external_$*/shim

external_%/shim: external_%/*.go
	go build -o external_$*/shim ./external_$*/

.PHONY: \
	test \
	lint
