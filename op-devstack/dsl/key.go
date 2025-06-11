package dsl

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// Key is an ethereum private key.
// This is a key with an address identity.
// The Key may be used on different chains: it is chain-agnostic.
type Key struct {
	t    devtest.T
	priv *ecdsa.PrivateKey
	addr common.Address
}

func NewKey(t devtest.T, priv *ecdsa.PrivateKey) *Key {
	t.Require().NotNil(priv.PublicKey, "private key PublicKey attribute must be initialized")
	addr := crypto.PubkeyToAddress(priv.PublicKey)
	return &Key{
		t:    t,
		priv: priv,
		addr: addr,
	}
}

func (a *Key) Priv() *ecdsa.PrivateKey {
	return a.priv
}

func (a *Key) String() string {
	return fmt.Sprintf("EOA(%s)", a.addr)
}

func (a *Key) Address() common.Address {
	return a.addr
}

// Plan returns the tx-plan option to use this Key for signing of a transaction.
func (a *Key) Plan() txplan.Option {
	return txplan.WithPrivateKey(a.priv)
}

// EOA combines this Key with an EL node into a single-chain EOA.
func (a *Key) User(el ELNode) *EOA {
	return NewEOA(a, el)
}
