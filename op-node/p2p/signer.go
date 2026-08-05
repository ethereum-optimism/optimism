package p2p

import (
	"context"
	"crypto/ecdsa"

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

// LocalSignerSetup creates a fresh signer for each node lifecycle. Unlike a
// PreparedSigner, it can be reused after a prior node closes its signer.
type LocalSignerSetup struct {
	privateKey *ecdsa.PrivateKey
}

func NewLocalSignerSetup(privateKey *ecdsa.PrivateKey) *LocalSignerSetup {
	return &LocalSignerSetup{privateKey: privateKey}
}

func (s *LocalSignerSetup) SetupSigner(context.Context) (Signer, error) {
	return opsigner.NewLocalSigner(s.privateKey), nil
}
