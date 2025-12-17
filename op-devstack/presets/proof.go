package presets

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

func WithRespectedGameType(gameType gameTypes.GameType) stack.CommonOption {
	opts := WithProposerGameType(gameType)
	opts = stack.Combine(opts,
		stack.MakeCommon(sysgo.WithRespectedGameType(gameType)), // Set if sysgo is in use
		RequireRespectedGameType(gameType),
	)
	return opts
}

func WithAddedGameType(gameType gameTypes.GameType) stack.CommonOption {
	opts := stack.Combine(
		stack.MakeCommon(sysgo.WithGameTypeAdded(gameType)), // Add if sysgo is in use
		RequireGameTypePresent(gameType),                    // Verify present for other chains
	)

	if gameType == gameTypes.CannonKonaGameType {
		opts = stack.Combine(
			opts,
			stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
		)
	}
	return opts
}

func RequireGameTypePresent(gameType gameTypes.GameType) stack.CommonOption {
	return stack.FnOption[stack.Orchestrator]{
		PostHydrateFn: func(sys stack.System) {
			elNode := sys.L1Network(match.FirstL1Network).L1ELNode(match.FirstL1EL)
			for _, l2Network := range sys.L2Networks() {
				dgf := bindings.NewBindings[bindings.DisputeGameFactory](
					bindings.WithClient(elNode.EthClient()),
					bindings.WithTo(l2Network.Deployment().DisputeGameFactoryProxyAddr()),
					bindings.WithTest(sys.T()),
				)
				gameImpl := contract.Read(dgf.GameImpls(uint32(gameType)))
				sys.T().Gate().NotZerof(gameImpl, "Dispute game factory must have a game implementation for %s", gameType)
			}
		},
	}
}

func RequireRespectedGameType(gameType gameTypes.GameType) stack.CommonOption {
	return stack.FnOption[stack.Orchestrator]{
		PostHydrateFn: func(sys stack.System) {

			elNode := sys.L1Network(match.FirstL1Network).L1ELNode(match.FirstL1EL)
			for _, l2Network := range sys.L2Networks() {
				l1PortalAddr := l2Network.RollupConfig().DepositContractAddress
				l1Portal := bindings.NewBindings[bindings.OptimismPortal2](
					bindings.WithClient(elNode.EthClient()),
					bindings.WithTo(l1PortalAddr),
					bindings.WithTest(sys.T()))

				respectedGameType, err := contractio.Read(l1Portal.RespectedGameType(), sys.T().Ctx())
				sys.T().Require().NoError(err, "Failed to read respected game type")
				sys.T().Gate().EqualValuesf(gameType, respectedGameType, "Respected game type must be %s", gameType)
			}
		},
	}
}

func WithProposerGameType(gameType gameTypes.GameType) stack.CommonOption {
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
