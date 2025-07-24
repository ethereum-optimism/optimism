package noopseq

import (
	"context"

	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

type Config struct {
}

func (c *Config) Start(ctx context.Context, id seqtypes.SequencerID, opts *work.ServiceOpts) (work.Sequencer, error) {
	return &Sequencer{
		id:  id,
		log: opts.Log,
	}, nil
}
