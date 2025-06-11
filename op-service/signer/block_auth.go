package signer

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type BlockSigner interface {
	// SignBlockV1 signs a P2P block, with the V1 signing domain
	SignBlockV1(ctx context.Context, chainID eth.ChainID, payloadHash common.Hash) (sig eth.Bytes65, err error)
	// Close closes the signer
	Close() error
}

// Authenticator abstractly represents the contextual information and logic needed to verify a SignedObject.
// E.g. a block is verified with context of the chain-ID, the allowed signers, and a versioned signing domain.
// Other block types (future OP-Stack versions, extensions, or e.g. L1 test chains)
// may be verified differently, providing different authentication context.
type Authenticator interface {
}

// SignedObject abstractly represents an object of which the signature can be checked.
type SignedObject interface {
	// VerifySignature verifies the signature of the signed object,
	// and returns an error if the signature is not valid.
	VerifySignature(auth Authenticator) error
}

// OPStackP2PBlockAuth provides a P2P block authenticator.
type OPStackP2PBlockAuth interface {
	// Check if the given signer is allowed to send messages
	Check(signer common.Address) error
	// Domain used to sign the message
	Domain() [32]byte
	// ChainID used to sign the message
	ChainID() eth.ChainID
	// VerifyP2PBlockSignature verifies a block with payload-hash and signature
	VerifyP2PBlockSignature(payloadHash common.Hash, signature eth.Bytes65) error
}

// SigningDomainBlocksV1 is the original signing domain used for P2P OP-Stack blocks.
// This domain is a fully zeroed 32 bytes.
var SigningDomainBlocksV1 = [32]byte{}

// OPStackP2PBlockAuthV1 provides the V1 OP-Stack P2P block authentication context.
type OPStackP2PBlockAuthV1 struct {
	Allowed common.Address
	Chain   eth.ChainID
}

var _ OPStackP2PBlockAuth = (*OPStackP2PBlockAuthV1)(nil)

func (a *OPStackP2PBlockAuthV1) Check(signer common.Address) error {
	if a.Allowed == (common.Address{}) {
		return errors.New("missing signer address configuration")
	}
	if a.Allowed == signer {
		return nil
	}
	return errors.New("unrecognized signer")
}

func (a *OPStackP2PBlockAuthV1) Domain() [32]byte {
	return SigningDomainBlocksV1
}

func (a *OPStackP2PBlockAuthV1) ChainID() eth.ChainID {
	return a.Chain
}

func (a *OPStackP2PBlockAuthV1) VerifyP2PBlockSignature(payloadHash common.Hash, signature eth.Bytes65) error {
	msg := BlockSigningMessage{
		Domain:      a.Domain(),
		ChainID:     a.ChainID(),
		PayloadHash: payloadHash,
	}
	signingHash := msg.ToSigningHash()

	pub, err := crypto.SigToPub(signingHash[:], signature[:])
	if err != nil {
		return fmt.Errorf("failed to recover pubkey from signing hash %s and pubkey %s: %w",
			signingHash, signature, err)
	}
	addr := crypto.PubkeyToAddress(*pub)
	return a.Check(addr)
}
