package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2BatcherConfig struct {
	CommonConfig
	ID      stack.ComponentID
	ChainID eth.ChainID
	Client  client.RPC
}

type rpcL2Batcher struct {
	commonImpl
	id      stack.ComponentID
	chainID eth.ChainID
	client  *sources.BatcherAdminClient
}

var _ stack.L2Batcher = (*rpcL2Batcher)(nil)

func NewL2Batcher(cfg L2BatcherConfig) stack.L2Batcher {
	cfg.T = cfg.T.WithCtx(stack.ContextWithComponentID(cfg.T.Ctx(), cfg.ID))
	return &rpcL2Batcher{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		chainID:    cfg.ChainID,
		client:     sources.NewBatcherAdminClient(cfg.Client),
	}
}

func (r *rpcL2Batcher) ID() stack.ComponentID {
	return r.id
}

func (r *rpcL2Batcher) ChainID() eth.ChainID {
	return r.chainID
}

func (p *rpcL2Batcher) ActivityAPI() apis.BatcherActivity {
	return p.client
}
