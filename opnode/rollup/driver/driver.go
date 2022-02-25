package driver

import (
	"context"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l1"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/sync"
	"github.com/ethereum/go-ethereum/log"
)

type Driver struct {
	s *state
}

func NewDriver(cfg rollup.Config, l2 DriverAPI, l1 l1.Source, log log.Logger) *Driver {
	input := &inputImpl{
		chainSource: sync.NewChainSource(l1, l2, &cfg.Genesis),
	}
	output := &outputImpl{
		Config: cfg,
		dl:     l1,
		log:    log,
		rpc:    l2,
	}
	return &Driver{
		s: NewState(log, cfg, input, output),
	}
}

func (d *Driver) Start(ctx context.Context, l1Heads <-chan eth.HeadSignal) error {
	return d.s.Start(ctx, l1Heads)
}
func (d *Driver) Close() error {
	return d.s.Close()
}

type inputImpl struct {
	chainSource sync.ChainSource
}

func (i *inputImpl) L1Head(ctx context.Context) (eth.L1Node, error) {
	return i.chainSource.L1HeadNode(ctx)
}

func (i *inputImpl) L2Head(ctx context.Context) (eth.L2Node, error) {
	return i.chainSource.L2NodeByNumber(ctx, nil)

}

func (i *inputImpl) L1ChainWindow(ctx context.Context, base eth.BlockID) ([]eth.BlockID, error) {
	return sync.FindL1Range(ctx, i.chainSource, base)
}
