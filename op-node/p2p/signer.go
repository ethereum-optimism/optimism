package p2p

import (
	"context"

	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

type Signer interface {
	opsigner.BlockSigner
}

type PreparedSigner struct {
	Signer
}

func (p *PreparedSigner) SetupSigner(ctx context.Context) (Signer, error) {
	return p.Signer, nil
}

type SignerSetup interface {
	SetupSigner(ctx context.Context) (Signer, error)
}
