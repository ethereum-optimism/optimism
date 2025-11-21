package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2ProposerConfig struct {
	CommonConfig
	ID      stack.ComponentID
	ChainID eth.ChainID
	Client  client.RPC
}

type rpcL2Proposer struct {
	commonImpl
	id      stack.ComponentID
	client  client.RPC
	chainID eth.ChainID
}

var _ stack.L2Proposer = (*rpcL2Proposer)(nil)

func NewL2Proposer(cfg L2ProposerConfig) stack.L2Proposer {
	cfg.T = cfg.T.WithCtx(stack.ContextWithComponentID(cfg.T.Ctx(), cfg.ID))
	return &rpcL2Proposer{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		chainID:    cfg.ChainID,
	}
}

func (r *rpcL2Proposer) ID() stack.ComponentID {
	return r.id
}

func (r *rpcL2Proposer) ChainID() eth.ChainID {
	return r.chainID
}
