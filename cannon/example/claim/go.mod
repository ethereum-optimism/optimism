module claim

go 1.20

require github.com/ethereum-optimism/optimism v0.0.0

require (
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

replace github.com/ethereum-optimism/optimism v0.0.0 => ../../..
