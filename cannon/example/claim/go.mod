module claim

go 1.21

toolchain go1.21.1

require github.com/ethereum-optimism/optimism v0.0.0

require (
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

replace github.com/ethereum-optimism/optimism v0.0.0 => ../../..
