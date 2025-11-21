package shim

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2ChallengerConfig struct {
	CommonConfig
	ID      stack.ComponentID
	ChainID eth.ChainID
	Config  *config.Config
}

type rpcL2Challenger struct {
	commonImpl
	id      stack.ComponentID
	chainID eth.ChainID
	config  *config.Config
}

func (r *rpcL2Challenger) Config() *config.Config {
	return r.config
}

var _ stack.L2Challenger = (*rpcL2Challenger)(nil)

func NewL2Challenger(cfg L2ChallengerConfig) stack.L2Challenger {
	cfg.T = cfg.T.WithCtx(stack.ContextWithComponentID(cfg.T.Ctx(), cfg.ID))
	return &rpcL2Challenger{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		chainID:    cfg.ChainID,
		config:     cfg.Config,
	}
}

func (r *rpcL2Challenger) ID() stack.ComponentID {
	return r.id
}

func (r *rpcL2Challenger) ChainID() eth.ChainID {
	return r.chainID
}
