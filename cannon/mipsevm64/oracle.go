package mipsevm64

import "github.com/ethereum-optimism/optimism/cannon/run"

type PreimageOracle interface {
	Hint(v []byte)
	GetPreimage(k [32]byte) []byte
}

var _ PreimageOracle = (*run.ProcessPreimageOracle)(nil)
