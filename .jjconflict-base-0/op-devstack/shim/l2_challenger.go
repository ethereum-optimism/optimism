package shim

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type L2ChallengerConfig struct {
	CommonConfig
	ID     stack.L2ChallengerID
	Config *config.Config
}

type rpcL2Challenger struct {
	commonImpl
	id     stack.L2ChallengerID
	config *config.Config
}

func (r *rpcL2Challenger) Config() *config.Config {
	return r.config
}

var _ stack.L2Challenger = (*rpcL2Challenger)(nil)

func NewL2Challenger(cfg L2ChallengerConfig) stack.L2Challenger {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &rpcL2Challenger{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		config:     cfg.Config,
	}
}

func (r *rpcL2Challenger) ID() stack.L2ChallengerID {
	return r.id
}
