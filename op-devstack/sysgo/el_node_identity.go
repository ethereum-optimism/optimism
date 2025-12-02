package sysgo

import (
	"crypto/ecdsa"
	"encoding/hex"
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type ELNodeIdentity struct {
	Key   *ecdsa.PrivateKey
	Port  int
	Enode string
}

func NewELNodeIdentity(addr string, port int) *ELNodeIdentity {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	if port <= 0 {
		portStr, err := getAvailableLocalPort()
		if err != nil {
			panic(err)
		}
		port, err = strconv.Atoi(portStr)
		if err != nil {
			panic(err)
		}
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		panic("invalid ip for ELNodeIdentity: " + addr)
	}
	return &ELNodeIdentity{
		Key:   key,
		Port:  port,
		Enode: enode.NewV4(&key.PublicKey, ip, port, port).String(),
	}
}

func (id *ELNodeIdentity) KeyHex() string {
	return hex.EncodeToString(crypto.FromECDSA(id.Key))
}
