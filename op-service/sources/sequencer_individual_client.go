package sources

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type IndividualClient struct {
	client client.RPC
}

var _ apis.TestSequencerIndividualAPI = (*IndividualClient)(nil)

func NewIndividualClient(client client.RPC) *IndividualClient {
	return &IndividualClient{
		client: client,
	}
}

func (sc *IndividualClient) BuildJob() (result seqtypes.BuildJobID, err error) {
	err = sc.client.CallContext(context.Background(), &result, "sequencer_buildJob")
	if err != nil {
		return result, fmt.Errorf("failed to build job for individual sequencer: %w", err)
	}
	return
}

func (sc *IndividualClient) Stop(ctx context.Context) (result common.Hash, err error) {
	err = sc.client.CallContext(ctx, &result, "sequencer_stop")
	return result, err
}

func (sc *IndividualClient) IncludeTx(ctx context.Context, tx hexutil.Bytes) error {
	err := sc.client.CallContext(ctx, nil, "sequencer_includeTx", tx)
	if err != nil {
		return fmt.Errorf("failed to include tx for Individual Sequencer: %w", err)
	}
	return nil
}

func (sc *IndividualClient) Start(ctx context.Context, head common.Hash) error {
	err := sc.client.CallContext(ctx, nil, "sequencer_start", head)
	if err != nil {
		return fmt.Errorf("failed to start Individual Sequencer: %w", err)
	}
	return nil
}

func (sc *IndividualClient) PrebuiltEnvelope(ctx context.Context, block *eth.ExecutionPayloadEnvelope) error {
	return sc.client.CallContext(ctx, nil, "sequencer_prebuiltEnvelope", block)
}

func (sc *IndividualClient) Commit(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_commit")
}

func (sc *IndividualClient) New(ctx context.Context, opts seqtypes.BuildOpts) error {
	return sc.client.CallContext(ctx, nil, "sequencer_new", opts)
}

func (sc *IndividualClient) Open(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_open")
}

func (sc *IndividualClient) Next(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_next")
}

func (sc *IndividualClient) Publish(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_publish")
}

func (sc *IndividualClient) Seal(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_seal")
}

func (sc *IndividualClient) Sign(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_sign")
}
