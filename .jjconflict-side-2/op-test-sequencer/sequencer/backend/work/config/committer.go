package config

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/noopcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/standardcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type CommitterEntry struct {
	Standard *standardcommitter.Config `yaml:"standard,omitempty"`
	Noop     *noopcommitter.Config     `yaml:"noop,omitempty"`
}

func (b *CommitterEntry) Start(ctx context.Context, id seqtypes.CommitterID, opts *work.ServiceOpts) (work.Committer, error) {
	switch {
	case b.Standard != nil:
		return b.Standard.Start(ctx, id, opts)
	case b.Noop != nil:
		return b.Noop.Start(ctx, id, opts)
	default:
		return nil, seqtypes.ErrUnknownKind
	}
}
