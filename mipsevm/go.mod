module mipsevm

go 1.17

replace github.com/ethereum/go-ethereum => ../minigeth

replace github.com/unicorn-engine/unicorn => ../unicorn

require (
	github.com/ethereum/go-ethereum v1.10.8
	github.com/fatih/color v1.13.0
	github.com/unicorn-engine/unicorn v0.0.0-20211005173419-3fadb5aa5aad
)

require (
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/holiman/uint256 v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
)
