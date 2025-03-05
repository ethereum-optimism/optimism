package config

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/noopsigner"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type SignerEntry struct {
	Noop *noopsigner.Config `yaml:"noop,omitempty"`
}

func (b *SignerEntry) Start(ctx context.Context, id seqtypes.SignerID, opts *work.ServiceOpts) (work.Signer, error) {
	switch {
	case b.Noop != nil:
		return b.Noop.Start(ctx, id, opts)
	default:
		return nil, seqtypes.ErrUnknownKind
	}
}
