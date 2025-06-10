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

// TODO(infra#401): Implement support in the sysext toolset
func WithFastGame() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithDeployerOptions(
			sysgo.WithAdditionalDisputeGames(
				[]state.AdditionalDisputeGame{
					{
						ChainProofParams: state.ChainProofParams{
							DisputeGameType: uint32(faultTypes.FastGameType),
							// Use Alphabet VM prestate which is a pre-determined fixed hash
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

// TODO(infra#401): Implement support in the sysext toolset
func WithDeployerMatchL1PAO() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithDeployerPipelineOption(
			sysgo.WithDeployerMatchL1PAO(),
		),
	)
}

// TODO(infra#401): Implement support in the sysext toolset
func WithGuardianMatchL1PAO() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithDeployerOptions(
			sysgo.WithGuardianMatchL1PAO(),
		),
	)
}

// TODO(infra#401): Implement support in the sysext toolset
func WithFinalizationPeriodSeconds(n uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		sysgo.WithFinalizationPeriodSeconds(n),
	))
}

// TODO(infra#401): Implement support in the sysext toolset
func WithProofMaturityDelaySeconds(seconds uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		sysgo.WithProofMaturityDelaySeconds(seconds),
	))
}

// TODO(infra#401): Implement support in the sysext toolset
func WithDisputeGameFinalityDelaySeconds(seconds uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		sysgo.WithDisputeGameFinalityDelaySeconds(seconds),
	))
}
