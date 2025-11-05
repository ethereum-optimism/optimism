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

// WithProofs enables a minimal system with permissionless proofs enabled
func WithProofs() stack.CommonOption {
	return stack.MakeCommon(sysgo.ProofSystem(&sysgo.DefaultMinimalSystemIDs{}))
}
