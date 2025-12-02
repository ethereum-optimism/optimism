package signer

import (
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SignedP2PBlock represents an untrusted block payload,
// of which the signature can be verified with VerifySignature
type SignedP2PBlock struct {
	Raw       hexutil.Bytes `json:"raw"`
	Signature eth.Bytes65   `json:"signature"`
}

func (s *SignedP2PBlock) VerifySignature(auth Authenticator) error {
	p2pAuth, ok := auth.(OPStackP2PBlockAuth)
	if !ok {
		return errors.New("expected P2P auth context")
	}
	payloadHash := PayloadHash(s.Raw)
	return p2pAuth.VerifyP2PBlockSignature(payloadHash, s.Signature)
}
