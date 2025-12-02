package frontend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type SequencerFrontend struct {
	Sequencer work.Sequencer
	Logger    log.Logger
}

func (bf *SequencerFrontend) New(ctx context.Context, opts seqtypes.BuildOpts) error {
	bf.Logger.Debug("SequencerFrontend New request", "parent", opts.Parent, "l1_origin", opts.L1Origin)

	return toJsonError(bf.Sequencer.New(ctx, opts))
}

func (bf *SequencerFrontend) BuildJob() (seqtypes.BuildJobID, error) {
	job := bf.Sequencer.BuildJob()
	if job == nil {
		return "", toJsonError(seqtypes.ErrUnknownJob)
	}
	return job.ID(), nil
}

func (bf *SequencerFrontend) Open(ctx context.Context) error {
	return toJsonError(bf.Sequencer.Open(ctx))
}

func (bf *SequencerFrontend) Seal(ctx context.Context) error {
	return toJsonError(bf.Sequencer.Seal(ctx))
}

func (bf *SequencerFrontend) PrebuiltEnvelope(ctx context.Context, block *eth.ExecutionPayloadEnvelope) error {
	return toJsonError(bf.Sequencer.Prebuilt(ctx, block))
}

func (bf *SequencerFrontend) Sign(ctx context.Context) error {
	return toJsonError(bf.Sequencer.Sign(ctx))
}

func (bf *SequencerFrontend) Commit(ctx context.Context) error {
	return toJsonError(bf.Sequencer.Commit(ctx))
}

func (bf *SequencerFrontend) Publish(ctx context.Context) error {
	return toJsonError(bf.Sequencer.Publish(ctx))
}

func (bf *SequencerFrontend) Next(ctx context.Context) error {
	return toJsonError(bf.Sequencer.Next(ctx))
}

func (bf *SequencerFrontend) Start(ctx context.Context, head common.Hash) error {
	return toJsonError(bf.Sequencer.Start(ctx, head))
}

func (bf *SequencerFrontend) Stop(ctx context.Context) (last common.Hash, err error) {
	last, err = bf.Sequencer.Stop(ctx)
	if err != nil {
		return common.Hash{}, toJsonError(err)
	}
	return
}

func (bf *SequencerFrontend) IncludeTx(ctx context.Context, tx hexutil.Bytes) error {
	job := bf.Sequencer.BuildJob()
	if job == nil {
		return toJsonError(seqtypes.ErrUnknownJob)
	}
	return toJsonError(job.IncludeTx(ctx, tx))
}
