package standardcommitter

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

// Committer commits blocks with the op-stack commit API.
// This can be an op-node that processes and persists the block as canonical.
type Committer struct {
	api     apis.CommitAPI
	onClose func()

	id  seqtypes.CommitterID
	log log.Logger
}

var _ work.Committer = (*Committer)(nil)

func NewCommitter(id seqtypes.CommitterID, log log.Logger, api apis.CommitAPI) *Committer {
	return &Committer{id: id, log: log, api: api, onClose: func() {}}
}

func (n *Committer) Close() error {
	n.onClose()
	return nil
}

func (n *Committer) String() string {
	return "standard-committer-" + n.id.String()
}

func (n *Committer) ID() seqtypes.CommitterID {
	return n.id
}

func (n *Committer) Commit(ctx context.Context, block work.SignedBlock) error {
	bl, ok := block.(*opsigner.SignedExecutionPayloadEnvelope)
	if !ok {
		return fmt.Errorf("cannot commit block of type %T: %w", block, seqtypes.ErrUnknownKind)
	}
	err := n.api.CommitBlock(ctx, bl)
	if err != nil {
		n.log.Error("Failed to publish block", "block", block, "err", err)
	}
	return err
}
