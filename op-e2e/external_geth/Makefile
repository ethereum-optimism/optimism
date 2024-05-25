default: shim op-geth

op-geth:
	go build -o op-geth "github.com/ethereum/go-ethereum/cmd/geth"
.PHONY: op-geth

shim: main.go
	go build -o shim .
