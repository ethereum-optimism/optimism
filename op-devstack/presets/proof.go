package presets

import (
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
)

func WithProposerGameType(gameType faultTypes.GameType) stack.CommonOption {
	return stack.Combine(
		stack.MakeCommon(
			sysgo.WithProposerOption(func(id stack.L2ProposerID, cfg *ps.CLIConfig) {
				cfg.DisputeGameType = uint32(gameType)
			})))
}
