package signer

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// LocalSigner is suitable for testing
type LocalSigner struct {
	priv *ecdsa.PrivateKey
}

var _ BlockSigner = (*LocalSigner)(nil)

func NewLocalSigner(priv *ecdsa.PrivateKey) *LocalSigner {
	return &LocalSigner{priv: priv}
}

func (s *LocalSigner) SignBlockV1(ctx context.Context, chainID eth.ChainID, payloadHash common.Hash) (sig eth.Bytes65, err error) {
	if s.priv == nil {
		return eth.Bytes65{}, errors.New("signer is closed")
	}
	msg := BlockSigningMessage{
		Domain:      SigningDomainBlocksV1,
		ChainID:     chainID,
		PayloadHash: payloadHash,
	}
	signingHash := msg.ToSigningHash()
	signature, err := crypto.Sign(signingHash[:], s.priv)
	if err != nil {
		return eth.Bytes65{}, err
	}
	return eth.Bytes65(signature), nil
}

func (s *LocalSigner) Close() error {
	s.priv = nil
	return nil
}
