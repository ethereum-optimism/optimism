package mipsevm

type PreimageOracle interface {
	Hint(v []byte)
	GetPreimage(k [32]byte) []byte
}
