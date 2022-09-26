package p2p

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var SigningDomainBlocksV1 = [32]byte{}

type Signer interface {
	Sign(ctx context.Context, domain [32]byte, chainID *big.Int, encodedMsg []byte) (sig *[65]byte, err error)
	io.Closer
}

func SigningHash(domain [32]byte, chainID *big.Int, payloadBytes []byte) common.Hash {
	var msgInput [32 + 32 + 32]byte
	// domain: first 32 bytes
	copy(msgInput[:32], domain[:])
	// chain_id: second 32 bytes
	chainID.FillBytes(msgInput[32:64])
	// payload_hash: third 32 bytes, hash of encoded payload
	copy(msgInput[32:], crypto.Keccak256(payloadBytes))

	return crypto.Keccak256Hash(msgInput[:])
}

func BlockSigningHash(cfg *rollup.Config, payloadBytes []byte) common.Hash {
	return SigningHash(SigningDomainBlocksV1, cfg.L2ChainID, payloadBytes)
}

// LocalSigner is suitable for testing
type LocalSigner struct {
	priv *ecdsa.PrivateKey
}

func NewLocalSigner(priv *ecdsa.PrivateKey) *LocalSigner {
	return &LocalSigner{priv: priv}
}

func (s *LocalSigner) Sign(ctx context.Context, domain [32]byte, chainID *big.Int, encodedMsg []byte) (sig *[65]byte, err error) {
	if s.priv == nil {
		return nil, errors.New("signer is closed")
	}
	signingHash := SigningHash(domain, chainID, encodedMsg)
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

// TODO: implement remote signer setup (config to authenticated endpoint)
// and remote signer itself (e.g. a open http client to make signing requests)

type SignerSetup interface {
	SetupSigner(ctx context.Context) (Signer, error)
}

// LoadSignerSetup loads a configuration for a Signer to be set up later
func LoadSignerSetup(ctx *cli.Context) (SignerSetup, error) {
	key := ctx.GlobalString(flags.SequencerP2PKeyFlag.Name)
	if key != "" {
		// Mnemonics are bad because they leak *all* keys when they leak.
		// Unencrypted keys from file are bad because they are easy to leak (and we are not checking file permissions).
		priv, err := crypto.HexToECDSA(key)
		if err != nil {
			return nil, fmt.Errorf("failed to read batch submitter key: %w", err)
		}

		return &PreparedSigner{Signer: NewLocalSigner(priv)}, nil
	}

	// TODO: create remote signer

	return nil, nil
}
