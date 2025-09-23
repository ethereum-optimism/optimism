package engine

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// ResetEngineRequestEvent requests the EngineResetDeriver to walk
// the L2 chain backwards until it finds a plausible unsafe head,
// and find an L2 safe block that is guaranteed to still be from the L1 chain.
// This event is not used in interop.
type ResetEngineRequestEvent struct {
}

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

	emitter event.Emitter

	engController *EngineController
}

func NewEngineResetDeriver(ctx context.Context, log log.Logger, cfg *rollup.Config,
	l1 sync.L1Chain, l2 sync.L2Chain, syncCfg *sync.Config) *EngineResetDeriver {
	return &EngineResetDeriver{
		ctx:     ctx,
		log:     log,
		cfg:     cfg,
		l1:      l1,
		l2:      l2,
		syncCfg: syncCfg,
	}
}

func (d *EngineResetDeriver) SetEngController(engController *EngineController) {
	d.engController = engController
}

func (d *EngineResetDeriver) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

func (d *EngineResetDeriver) OnEvent(ctx context.Context, ev event.Event) bool {
	switch ev.(type) {
	case ResetEngineRequestEvent:
		result, err := sync.FindL2Heads(d.ctx, d.cfg, d.l1, d.l2, d.log, d.syncCfg)
		if err != nil {
			d.emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("failed to find the L2 Heads to start from: %w", err),
			})
			return true
		}
		d.engController.ForceReset(ctx, result.Unsafe, result.Unsafe, result.Safe, result.Safe, result.Finalized)
	default:
		return false
	}
	return true
}
