package sources

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type BuilderClient struct {
	client client.RPC
}

var _ apis.SequencerAPI = (*BuilderClient)(nil)

func NewBuilderClient(client client.RPC) *BuilderClient {
	return &BuilderClient{
		client: client,
	}
}

func (sc *BuilderClient) Stop(ctx context.Context) error {
	err := sc.client.CallContext(ctx, nil, "admin_stop")
	if err != nil {
		return fmt.Errorf("failed to stop Sequencer: %w", err)
	}
	return nil
}

func (sc *BuilderClient) Start(ctx context.Context) error {
	err := sc.client.CallContext(ctx, nil, "admin_start")
	if err != nil {
		return fmt.Errorf("failed to start Sequencer: %w", err)
	}
	return nil
}

func (sc *BuilderClient) Close() {
	sc.client.Close()
}

func (sc *BuilderClient) New(ctx context.Context, builderID seqtypes.BuilderID, opts *seqtypes.BuildOpts) (result seqtypes.BuildJobID, err error) {
	err = sc.client.CallContext(ctx, &result, "build_new", builderID, opts)
	return result, err
}

func (sc *BuilderClient) Open(ctx context.Context, jobID seqtypes.BuildJobID) error {
	return sc.client.CallContext(ctx, nil, "build_open", jobID)
}

func (sc *BuilderClient) Cancel(ctx context.Context, jobID seqtypes.BuildJobID) error {
	return sc.client.CallContext(ctx, nil, "build_cancel", jobID)
}

func (sc *BuilderClient) Seal(ctx context.Context, jobID seqtypes.BuildJobID) (result work.Block, err error) {
	err = sc.client.CallContext(ctx, &result, "build_seal", jobID)
	return result, err
}

func (sc *BuilderClient) CloseJob(id seqtypes.BuildJobID) error {
	return sc.client.CallContext(context.Background(), nil, "build_closeJob", id)
}
