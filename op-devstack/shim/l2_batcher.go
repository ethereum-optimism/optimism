package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// pauseController is a minimal interface for pause control, avoiding circular dependencies
type pauseController interface {
	PauseAtBlock(blockNum uint64) uint64
	Unpause()
	IsPaused() (bool, uint64)
}

type L2BatcherConfig struct {
	CommonConfig
	ID      stack.L2BatcherID
	Client  client.RPC
	Backend pauseController // Optional: only set for sysgo backend with test control
}

type rpcL2Batcher struct {
	commonImpl
	id      stack.L2BatcherID
	client  *sources.BatcherAdminClient
	backend pauseController // Optional: for test control access
}

var _ stack.L2Batcher = (*rpcL2Batcher)(nil)

func NewL2Batcher(cfg L2BatcherConfig) stack.L2Batcher {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &rpcL2Batcher{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     sources.NewBatcherAdminClient(cfg.Client),
		backend:    cfg.Backend,
	}
}

func (r *rpcL2Batcher) ID() stack.L2BatcherID {
	return r.id
}

func (p *rpcL2Batcher) ActivityAPI() apis.BatcherActivity {
	return p.client
}

// PausableBatcher implementation - delegates to backend if available

func (r *rpcL2Batcher) PauseAtBlock(blockNum uint64) uint64 {
	if r.backend != nil {
		return r.backend.PauseAtBlock(blockNum)
	}
	return 0
}

func (r *rpcL2Batcher) Unpause() {
	if r.backend != nil {
		r.backend.Unpause()
	}
}

func (r *rpcL2Batcher) IsPaused() (bool, uint64) {
	if r.backend != nil {
		return r.backend.IsPaused()
	}
	return false, 0
}
