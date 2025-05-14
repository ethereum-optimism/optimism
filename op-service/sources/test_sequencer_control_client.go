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

type ControlClient struct {
	client client.RPC
}

var _ apis.TestSequencerControlAPI = (*ControlClient)(nil)

func NewControlClient(client client.RPC) *ControlClient {
	return &ControlClient{
		client: client,
	}
}

func (sc *ControlClient) BuildJob() (result seqtypes.BuildJobID, err error) {
	err = sc.client.CallContext(context.Background(), &result, "sequencer_buildJob")
	if err != nil {
		return result, fmt.Errorf("failed to build job for individual sequencer: %w", err)
	}
	return
}

func (sc *ControlClient) Stop(ctx context.Context) (result common.Hash, err error) {
	err = sc.client.CallContext(ctx, &result, "sequencer_stop")
	return result, err
}

func (sc *ControlClient) IncludeTx(ctx context.Context, tx hexutil.Bytes) error {
	err := sc.client.CallContext(ctx, nil, "sequencer_includeTx", tx)
	if err != nil {
		return fmt.Errorf("failed to include tx for Individual Sequencer: %w", err)
	}
	return nil
}

func (sc *ControlClient) Start(ctx context.Context, head common.Hash) error {
	err := sc.client.CallContext(ctx, nil, "sequencer_start", head)
	if err != nil {
		return fmt.Errorf("failed to start Individual Sequencer: %w", err)
	}
	return nil
}

func (sc *ControlClient) PrebuiltEnvelope(ctx context.Context, block *eth.ExecutionPayloadEnvelope) error {
	return sc.client.CallContext(ctx, nil, "sequencer_prebuiltEnvelope", block)
}

func (sc *ControlClient) Commit(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_commit")
}

func (sc *ControlClient) New(ctx context.Context, opts seqtypes.BuildOpts) error {
	return sc.client.CallContext(ctx, nil, "sequencer_new", opts)
}

func (sc *ControlClient) Open(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_open")
}

func (sc *ControlClient) Next(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_next")
}

func (sc *ControlClient) Publish(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_publish")
}

func (sc *ControlClient) Seal(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_seal")
}

func (sc *ControlClient) Sign(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "sequencer_sign")
}
