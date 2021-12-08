module mipsevm

go 1.17

replace github.com/ethereum/go-ethereum => ../minigeth

replace github.com/unicorn-engine/unicorn => ../unicorn2

require (
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/ethereum/go-ethereum v1.10.8
	github.com/fatih/color v1.13.0
	github.com/unicorn-engine/unicorn v0.0.0-20211005173419-3fadb5aa5aad
)
