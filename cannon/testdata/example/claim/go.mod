module claim

go 1.22

toolchain go1.22.0

require github.com/ethereum-optimism/optimism v0.0.0

require (
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
)

replace github.com/ethereum-optimism/optimism v0.0.0 => ../../../..
