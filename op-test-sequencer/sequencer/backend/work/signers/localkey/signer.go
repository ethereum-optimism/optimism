package localkey

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type Signer struct {
	id  seqtypes.SignerID
	log log.Logger

	chainID eth.ChainID
	signer  *opsigner.LocalSigner
}

var _ work.Signer = (*Signer)(nil)

func NewSigner(id seqtypes.SignerID, log log.Logger, chainID eth.ChainID, priv *ecdsa.PrivateKey) *Signer {
	s := opsigner.NewLocalSigner(priv)
	return &Signer{id: id, log: log, chainID: chainID, signer: s}
}

func (s *Signer) String() string {
	return "local-key-signer-" + s.id.String()
}

func (s *Signer) ID() seqtypes.SignerID {
	return s.id
}

func (s *Signer) Close() error {
	return nil
}

func (s *Signer) Sign(ctx context.Context, v work.Block) (work.SignedBlock, error) {
	envelope, ok := v.(*eth.ExecutionPayloadEnvelope)
	if !ok {
		return nil, fmt.Errorf("cannot sign unknown block kind %T: %w", v, seqtypes.ErrUnknownKind)
	}

	var buf bytes.Buffer
	if _, err := envelope.MarshalSSZ(&buf); err != nil {
		return nil, fmt.Errorf("failed to encode execution payload: %w", err)
	}
	payloadHash := opsigner.PayloadHash(buf.Bytes())
	sig, err := s.signer.SignBlockV1(ctx, s.chainID, payloadHash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with local key: %w", err)
	}
	return &opsigner.SignedExecutionPayloadEnvelope{
		Envelope:  envelope,
		Signature: sig,
	}, nil
}

func (s *Signer) ChainID() eth.ChainID {
	return s.chainID
}
