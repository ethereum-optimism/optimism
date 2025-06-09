package presets

import (
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum/go-ethereum/common"
)

func WithProposerGameType(gameType faultTypes.GameType) stack.CommonOption {
	return stack.Combine(
		stack.MakeCommon(
			sysgo.WithProposerOption(func(id stack.L2ProposerID, cfg *ps.CLIConfig) {
				cfg.DisputeGameType = uint32(gameType)
			})))
}

func WithFastGame() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithDeployerOptions(
			sysgo.WithAdditionalDisputeGames(
				[]state.AdditionalDisputeGame{
					{
						ChainProofParams: state.ChainProofParams{
							DisputeGameType:         uint32(faultTypes.FastGameType),
							DisputeAbsolutePrestate: common.HexToHash("0x03c7ae758795765c6664a5d39bf63841c71ff191e9189522bad8ebff5d4eca98"),
							DisputeMaxGameDepth:     14 + 3 + 1,
							DisputeSplitDepth:       14,
							DisputeClockExtension:   0,
							DisputeMaxClockDuration: 0,
						},
						VMType:                       state.VMTypeAlphabet,
						UseCustomOracle:              true,
						OracleMinProposalSize:        10000,
						OracleChallengePeriodSeconds: 0,
						MakeRespected:                true,
					},
				},
			),
		),
	)
}

func WithDeployerMatchL1PAO() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithDeployerPipelineOption(
			sysgo.WithDeployerMatchL1PAO(),
		),
	)
}

func WithGuardianMatchL1PAO() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithDeployerOptions(
			sysgo.WithGuardianMatchL1PAO(),
		),
	)
}

func WithFinalizationPeriodSeconds(n uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		sysgo.WithFinalizationPeriodSeconds(n),
	))
}

func WithProofMaturityDelaySeconds(seconds uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		sysgo.WithProofMaturityDelaySeconds(seconds),
	))
}

func WithDisputeGameFinalityDelaySeconds(seconds uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		sysgo.WithDisputeGameFinalityDelaySeconds(seconds),
	))
}
