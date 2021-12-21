package secp256k1

import "github.com/ethereum/go-ethereum/crypto/secp256k1"

// S256 returns a BitCurve which implements secp256k1.
func S256() *secp256k1.BitCurve {
	return secp256k1.S256()
}
