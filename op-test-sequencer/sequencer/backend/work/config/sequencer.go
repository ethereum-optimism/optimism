package config

import (
	"context"

	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/sequencers/fullseq"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/sequencers/noopseq"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

type SequencerEntry struct {
	Full *fullseq.Config `yaml:"full,omitempty"`
	Noop *noopseq.Config `yaml:"noop,omitempty"`
}

func (b *SequencerEntry) Start(ctx context.Context, id seqtypes.SequencerID, opts *work.ServiceOpts) (work.Sequencer, error) {
	switch {
	case b.Full != nil:
		return b.Full.Start(ctx, id, opts)
	case b.Noop != nil:
		return b.Noop.Start(ctx, id, opts)
	default:
		return nil, seqtypes.ErrUnknownKind
	}
}
