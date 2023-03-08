package p2p

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var SigningDomainBlocksV1 = [32]byte{}

type Signer interface {
	Sign(ctx context.Context, domain [32]byte, chainID *big.Int, encodedMsg []byte) (sig *[65]byte, err error)
	io.Closer
}

func LegacySigningHash(domain [32]byte, chainID *big.Int, payloadBytes []byte) (common.Hash, error) {
	var msgInput [32 + 32 + 32]byte
	// domain: first 32 bytes
	copy(msgInput[:32], domain[:])
	// chain_id: second 32 bytes
	if chainID.BitLen() > 256 {
		return common.Hash{}, errors.New("chain_id is too large")
	}
	chainID.FillBytes(msgInput[32:64])
	// payload_hash: third 32 bytes, hash of encoded payload
	copy(msgInput[32:], crypto.Keccak256(payloadBytes))

	return crypto.Keccak256Hash(msgInput[:]), nil
}

func SigningHash(domain [32]byte, chainID *big.Int, payloadBytes []byte) (common.Hash, error) {
	var msgInput [32 + 32 + 32]byte
	// domain: first 32 bytes
	copy(msgInput[:32], domain[:])
	// chain_id: second 32 bytes
	if chainID.BitLen() > 256 {
		return common.Hash{}, errors.New("chain_id is too large")
	}
	chainID.FillBytes(msgInput[32:64])
	// payload_hash: third 32 bytes, hash of encoded payload
	copy(msgInput[64:], crypto.Keccak256(payloadBytes))

	return crypto.Keccak256Hash(msgInput[:]), nil
}

func BlockSigningHash(cfg *rollup.Config, payloadBytes []byte) (common.Hash, error) {
	return SigningHash(SigningDomainBlocksV1, cfg.L2ChainID, payloadBytes)
}

func LegacyBlockSigningHash(cfg *rollup.Config, payloadBytes []byte) (common.Hash, error) {
	return LegacySigningHash(SigningDomainBlocksV1, cfg.L2ChainID, payloadBytes)
}

// LocalSigner is suitable for testing
type LocalSigner struct {
	priv   *ecdsa.PrivateKey
	hasher func(domain [32]byte, chainID *big.Int, payloadBytes []byte) (common.Hash, error)
}

func NewLocalSigner(priv *ecdsa.PrivateKey) *LocalSigner {
	return &LocalSigner{priv: priv, hasher: SigningHash}
}

func (s *LocalSigner) Sign(ctx context.Context, domain [32]byte, chainID *big.Int, encodedMsg []byte) (sig *[65]byte, err error) {
	if s.priv == nil {
		return nil, errors.New("signer is closed")
	}
	signingHash, err := s.hasher(domain, chainID, encodedMsg)
	if err != nil {
		return nil, err
	}
	signature, err := crypto.Sign(signingHash[:], s.priv)
	if err != nil {
		return nil, err
	}
	return (*[65]byte)(signature), nil
}

func (s *LocalSigner) Close() error {
	s.priv = nil
	return nil
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
