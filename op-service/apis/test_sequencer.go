package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type TestSequencerAPI interface {
	TestSequencerAdminAPI
	TestSequencerBuildAPI
}

type TestSequencerBuildAPI interface {
	Close()
	New(context.Context, seqtypes.BuilderID, *seqtypes.BuildOpts) (seqtypes.BuildJobID, error)
	Open(context.Context, seqtypes.BuildJobID) error
	Cancel(context.Context, seqtypes.BuildJobID) error
	Seal(context.Context, seqtypes.BuildJobID) (work.Block, error)
	CloseJob(seqtypes.BuildJobID) error
}

type TestSequencerAdminAPI interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type TestSequencerControlAPI interface {
	BuildJob() (seqtypes.BuildJobID, error)
	Commit(ctx context.Context) error
	IncludeTx(ctx context.Context, tx hexutil.Bytes) error
	Next(ctx context.Context) error
	New(ctx context.Context, opts seqtypes.BuildOpts) error
	PrebuiltEnvelope(ctx context.Context, block *eth.ExecutionPayloadEnvelope) error
	Publish(ctx context.Context) error
	Open(ctx context.Context) error
	Seal(ctx context.Context) error
	Sign(ctx context.Context) error
	Start(ctx context.Context, head common.Hash) error
	Stop(ctx context.Context) (common.Hash, error)
}
