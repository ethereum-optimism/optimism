package noopsigner

import (
	"context"

	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

type Config struct {
}

func (c *Config) Start(ctx context.Context, id seqtypes.SignerID, opts *work.ServiceOpts) (work.Signer, error) {
	return &Signer{
		id:  id,
		log: opts.Log,
	}, nil
}
