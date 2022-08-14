package p2p

import (
	"math/big"
	"testing"
)

func TestSigningHash(t *testing.T) {
	var domain = [32]byte{}
	payload := make([]byte, 32)

	var c big.Int
	c.Exp(big.NewInt(10), big.NewInt(99), nil) // 10e99
	chainId := &c

	SigningHash(domain, chainId, payload)
}
