package standardpublisher

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type Publisher struct {
	api     apis.PublishAPI
	onClose func()

	id  seqtypes.PublisherID
	log log.Logger
}

var _ work.Publisher = (*Publisher)(nil)

func NewPublisher(id seqtypes.PublisherID, log log.Logger, api apis.PublishAPI) *Publisher {
	return &Publisher{id: id, log: log, api: api, onClose: func() {}}
}

func (n *Publisher) Close() error {
	n.onClose()
	return nil
}

func (n *Publisher) String() string {
	return "standard-publisher-" + n.id.String()
}

func (n *Publisher) ID() seqtypes.PublisherID {
	return n.id
}

func (n *Publisher) Publish(ctx context.Context, block work.SignedBlock) error {
	bl, ok := block.(*opsigner.SignedExecutionPayloadEnvelope)
	if !ok {
		return fmt.Errorf("cannot publish block of type %T: %w", block, seqtypes.ErrUnknownKind)
	}
	err := n.api.PublishBlock(ctx, bl)
	if err != nil {
		n.log.Error("Failed to publish block", "block", block, "err", err)
	}
	return err
}
