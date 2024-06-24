package engine

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

// ResetEngineRequestEvent requests the EngineResetDeriver to walk
// the L2 chain backwards until it finds a plausible unsafe head,
// and find an L2 safe block that is guaranteed to still be from the L1 chain.
type ResetEngineRequestEvent struct{}

func (ev ResetEngineRequestEvent) String() string {
	return "reset-engine-request"
}

type EngineResetDeriver struct {
	ctx     context.Context
	log     log.Logger
	cfg     *rollup.Config
	l1      sync.L1Chain
	l2      sync.L2Chain
	syncCfg *sync.Config

	emitter rollup.EventEmitter
}

func NewEngineResetDeriver(ctx context.Context, log log.Logger, cfg *rollup.Config,
	l1 sync.L1Chain, l2 sync.L2Chain, syncCfg *sync.Config, emitter rollup.EventEmitter) *EngineResetDeriver {
	return &EngineResetDeriver{
		ctx:     ctx,
		log:     log,
		cfg:     cfg,
		l1:      l1,
		l2:      l2,
		syncCfg: syncCfg,
		emitter: emitter,
	}
}

func (d *EngineResetDeriver) OnEvent(ev rollup.Event) {
	switch ev.(type) {
	case ResetEngineRequestEvent:
		result, err := sync.FindL2Heads(d.ctx, d.cfg, d.l1, d.l2, d.log, d.syncCfg)
		if err != nil {
			d.emitter.Emit(rollup.ResetEvent{Err: fmt.Errorf("failed to find the L2 Heads to start from: %w", err)})
			return
		}
		d.emitter.Emit(ForceEngineResetEvent{
			Unsafe:    result.Unsafe,
			Safe:      result.Safe,
			Finalized: result.Finalized,
		})
	}
}
