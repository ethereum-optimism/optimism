package noopbuilder

import (
	"context"

	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

type Config struct {
}

func (c *Config) Start(ctx context.Context, id seqtypes.BuilderID, opts *work.ServiceOpts) (work.Builder, error) {
	return &Builder{
		id:       id,
		registry: opts.Jobs,
	}, nil
}
