package interop

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	CrossL2Outbox = common.HexToAddress("")
)

type PostieConfig struct {
	Postie          *ecdsa.PrivateKey
	ConnectedChains []ethclient.Client
}

type Postie struct{}

func NewPostie() *Postie {
	return &Postie{}
}

func (p *Postie) Start(ctx context.Context) error {
	return nil
}

func (p *Postie) Stop(ctx context.Context) error {
	return nil
}

func (p *Postie) Stopped() bool {
	return false
}
