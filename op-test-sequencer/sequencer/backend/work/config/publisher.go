package config

import (
	"context"

	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/publishers/nooppublisher"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/publishers/standardpublisher"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

type PublisherEntry struct {
	Standard *standardpublisher.Config `yaml:"standard,omitempty"`
	Noop     *nooppublisher.Config     `yaml:"noop,omitempty"`
}

func (b *PublisherEntry) Start(ctx context.Context, id seqtypes.PublisherID, opts *work.ServiceOpts) (work.Publisher, error) {
	switch {
	case b.Standard != nil:
		return b.Standard.Start(ctx, id, opts)
	case b.Noop != nil:
		return b.Noop.Start(ctx, id, opts)
	default:
		return nil, seqtypes.ErrUnknownKind
	}
}
