module github.com/ethereum-optimism/optimistic-specs

go 1.17

require (
	github.com/ethereum/go-ethereum v1.10.13
	github.com/protolambda/ask v0.1.2
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
)

require (
	github.com/go-stack/stack v1.8.0 // indirect
	golang.org/x/sys v0.0.0-20210816183151-1e6c022a8912 // indirect
)

replace github.com/ethereum/go-ethereum v1.10.13 => github.com/ethereum-optimism/reference-optimistic-geth v0.0.0-20220107224313-7f6d88bc156a

//replace github.com/ethereum/go-ethereum v1.10.13 => ../reference-optimistic-geth
