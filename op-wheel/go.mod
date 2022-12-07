module github.com/ethereum-optimism/optimism/op-wheel

go 1.18

require (
	github.com/ethereum-optimism/optimism/op-service v0.10.3
	github.com/ethereum/go-ethereum v1.10.26
	github.com/urfave/cli v1.22.10
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/term v0.3.0 // indirect
)

replace github.com/ethereum/go-ethereum v1.10.26 => github.com/ethereum-optimism/op-geth v0.0.0-20221205191237-0678a130d790
