module claim

go 1.22

toolchain go1.23.0

require github.com/ethereum-optimism/optimism v0.0.0

require (
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
)

replace github.com/ethereum-optimism/optimism v0.0.0 => ../../../..
