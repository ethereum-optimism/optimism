package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

type L2BatcherConfig struct {
	CommonConfig
	ID     stack.L2BatcherID
	Client client.RPC
}

type rpcL2Batcher struct {
	commonImpl
	id     stack.L2BatcherID
	client client.RPC
}

var _ stack.L2Batcher = (*rpcL2Batcher)(nil)

func NewL2Batcher(cfg L2BatcherConfig) stack.L2Batcher {
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL2Batcher{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
	}
}

func (r *rpcL2Batcher) ID() stack.L2BatcherID {
	return r.id
}
