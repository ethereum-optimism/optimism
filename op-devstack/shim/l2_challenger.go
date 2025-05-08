package shim

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

type L2ChallengerConfig struct {
	CommonConfig
	ID stack.L2ChallengerID
}

type rpcL2Challenger struct {
	commonImpl
	id stack.L2ChallengerID
}

var _ stack.L2Challenger = (*rpcL2Challenger)(nil)

func NewL2Challenger(cfg L2ChallengerConfig) stack.L2Challenger {
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL2Challenger{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
	}
}

func (r *rpcL2Challenger) ID() stack.L2ChallengerID {
	return r.id
}
