all: devnet

devnet:
	go build -o ./bin/devnet ./cmd/devnet/main.go

allocs:
	go build -o ./bin/allocs ./cmd/allocs/main.go

test:
	go test ./...

.PHONY: devnet allocs test fuzz
