package config

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/fakepos"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/noopbuilder"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/standardbuilder"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type BuilderEntry struct {
	Standard *standardbuilder.Config `yaml:"standard,omitempty"`
	Noop     *noopbuilder.Config     `yaml:"noop,omitempty"`
	L1       *fakepos.Config         // L1 is supported only in-process
}

func (b *BuilderEntry) Start(ctx context.Context, id seqtypes.BuilderID, opts *work.ServiceOpts) (work.Builder, error) {
	switch {
	case b.Standard != nil:
		return b.Standard.Start(ctx, id, opts)
	case b.Noop != nil:
		return b.Noop.Start(ctx, id, opts)
	case b.L1 != nil:
		return fakepos.NewBuilder(ctx, id, opts, b.L1)
	default:
		return nil, seqtypes.ErrUnknownKind
	}
}
