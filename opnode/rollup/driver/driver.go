package driver

import (
	"context"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l1"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/sync"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Driver struct {
	s *state
}

type BatchSubmitter interface {
	Submit(config *rollup.Config, batches []*derive.BatchData) (common.Hash, error)
}

func NewDriver(cfg rollup.Config, l2 DriverAPI, l1 l1.Source, log log.Logger, submitter BatchSubmitter, sequencer bool) *Driver {
	if sequencer && submitter == nil {
		log.Error("Bad configuration")
		// TODO: return error
	}
	input := &inputImpl{
		chainSource: sync.NewChainSource(l1, l2, &cfg.Genesis),
		genesis:     &cfg.Genesis,
	}
	output := &outputImpl{
		Config: cfg,
		dl:     l1,
		log:    log,
		rpc:    l2,
	}
	return &Driver{
		s: NewState(log, cfg, input, output, submitter, sequencer),
	}
}

func (d *Driver) Start(ctx context.Context, l1Heads <-chan eth.L1BlockRef) error {
	return d.s.Start(ctx, l1Heads)
}
func (d *Driver) Close() error {
	return d.s.Close()
}

type inputImpl struct {
	chainSource sync.ChainSource
	genesis     *rollup.Genesis
}

func (i *inputImpl) L1Head(ctx context.Context) (eth.L1BlockRef, error) {
	return i.chainSource.L1HeadBlockRef(ctx)
}

func (i *inputImpl) L2Head(ctx context.Context) (eth.L2BlockRef, error) {
	return i.chainSource.L2BlockRefByNumber(ctx, nil)

}

func (i *inputImpl) L1ChainWindow(ctx context.Context, base eth.BlockID) ([]eth.BlockID, error) {
	return sync.FindL1Range(ctx, i.chainSource, base)
}

func (i *inputImpl) SafeL2Head(ctx context.Context) (eth.L2BlockRef, error) {
	return sync.FindSafeL2Head(ctx, i.chainSource, i.genesis)
}
