module alloc

go 1.25

toolchain go1.25.4

require github.com/ethereum-optimism/optimism v0.0.0

require (
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
)

replace github.com/ethereum-optimism/optimism v0.0.0 => ./../../../..
