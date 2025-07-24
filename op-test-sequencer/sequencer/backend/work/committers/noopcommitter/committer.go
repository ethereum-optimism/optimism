package noopcommitter

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

type Committer struct {
	id  seqtypes.CommitterID
	log log.Logger
}

var _ work.Committer = (*Committer)(nil)

func NewCommitter(id seqtypes.CommitterID, log log.Logger) *Committer {
	return &Committer{id: id, log: log}
}

func (n *Committer) Close() error {
	return nil
}

func (n *Committer) String() string {
	return "noop-committer-" + n.id.String()
}

func (n *Committer) ID() seqtypes.CommitterID {
	return n.id
}

func (n *Committer) Commit(ctx context.Context, block work.SignedBlock) error {
	n.log.Info("No-op commit", "block", block)
	return nil
}
