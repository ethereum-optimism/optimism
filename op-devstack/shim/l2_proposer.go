package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

type L2ProposerConfig struct {
	CommonConfig
	ID     stack.L2ProposerID
	Client client.RPC
}

type rpcL2Proposer struct {
	commonImpl
	id     stack.L2ProposerID
	client client.RPC
}

var _ stack.L2Proposer = (*rpcL2Proposer)(nil)

func NewL2Proposer(cfg L2ProposerConfig) stack.L2Proposer {
	ctx := cfg.T.Ctx()
	ctx = stack.ContextWithKind(ctx, stack.L2ProposerKind)
	ctx = stack.ContextWithChainID(ctx, cfg.ID.ChainID)
	cfg.T = cfg.T.WithCtx(ctx, "chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL2Proposer{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
	}
}

func (r *rpcL2Proposer) ID() stack.L2ProposerID {
	return r.id
}
