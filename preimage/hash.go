package preimage

import "golang.org/x/crypto/sha3"

func Keccak256(v []byte) (out [32]byte) {
	s := sha3.NewLegacyKeccak256()
	s.Write(v)
	s.Sum(out[:0])
	return
}
