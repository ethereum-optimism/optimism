package sysgo

import (
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
)

type ELNodeIdentity struct {
	Key  *ecdsa.PrivateKey
	Port int
}

func NewELNodeIdentity(port int) *ELNodeIdentity {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return &ELNodeIdentity{
		Key:  key,
		Port: port,
	}
}

func (id *ELNodeIdentity) KeyHex() string {
	return hex.EncodeToString(crypto.FromECDSA(id.Key))
}
