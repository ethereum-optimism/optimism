package signer

import (
	"bytes"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SignedExecutionPayloadEnvelope struct {
	Envelope  *eth.ExecutionPayloadEnvelope `json:"envelope"`
	Signature eth.Bytes65                   `json:"signature"`
}

var _ SignedObject = (*SignedExecutionPayloadEnvelope)(nil)

func (s *SignedExecutionPayloadEnvelope) ID() eth.BlockID {
	return s.Envelope.ExecutionPayload.ID()
}

func (s *SignedExecutionPayloadEnvelope) String() string {
	return fmt.Sprintf("signedEnvelope(%s)", s.ID())
}

func (s *SignedExecutionPayloadEnvelope) VerifySignature(auth Authenticator) error {
	var buf bytes.Buffer
	if _, err := s.Envelope.MarshalSSZ(&buf); err != nil {
		return fmt.Errorf("failed to encode execution envelope: %w", err)
	}
	enc := SignedP2PBlock{
		Raw:       buf.Bytes(),
		Signature: s.Signature,
	}
	return enc.VerifySignature(auth)
}
