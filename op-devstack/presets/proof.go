package presets

import (
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

const (
	DefaultZKChallengeDuration = 30 * time.Minute
	DefaultZKProveDuration     = 30 * time.Minute
	DefaultZKProposalInterval  = 6 * time.Second
	DefaultZKFinalityDelay     = 2 * time.Second
)

type proofValidationTarget interface {
	proofValidationContext() (devtest.T, *dsl.L1ELNode, []*dsl.L2Network)
}

func afterBuildProofValidation(fn func(t devtest.T, elNode *dsl.L1ELNode, l2Networks []*dsl.L2Network)) Option {
	var kinds optionKinds
	if fn != nil {
		kinds = optionKindAfterBuild | optionKindProofValidation
	}
	return option{
		kinds: kinds,
		applyPresetFn: func(target any) {
			if fn == nil {
				return
			}
			proofTarget, ok := target.(proofValidationTarget)
			if !ok {
				return
			}
			t, elNode, l2Networks := proofTarget.proofValidationContext()
			fn(t, elNode, l2Networks)
		},
	}
}

func WithRespectedGameType(gameType gameTypes.GameType) Option {
	opts := WithProposerGameType(gameType)
	opts = Combine(opts,
		WithRespectedGameTypeOverride(gameType),
		RequireRespectedGameType(gameType),
	)
	return opts
}

func RequireGameTypePresent(gameType gameTypes.GameType) Option {
	return afterBuildProofValidation(func(t devtest.T, elNode *dsl.L1ELNode, l2Networks []*dsl.L2Network) {
		for _, l2Network := range l2Networks {
			dgf := bindings.NewBindings[bindings.DisputeGameFactory](
				bindings.WithClient(elNode.EthClient()),
				bindings.WithTo(l2Network.Escape().Deployment().DisputeGameFactoryProxyAddr()),
				bindings.WithTest(t),
			)
			gameImpl := contract.Read(dgf.GameImpls(uint32(gameType)))
			t.Gate().NotZerof(gameImpl, "Dispute game factory must have a game implementation for %s", gameType)
		}
	})
}

func RequireRespectedGameType(gameType gameTypes.GameType) Option {
	return afterBuildProofValidation(func(t devtest.T, elNode *dsl.L1ELNode, l2Networks []*dsl.L2Network) {
		for _, l2Network := range l2Networks {
			l1PortalAddr := l2Network.Escape().RollupConfig().DepositContractAddress
			l1Portal := bindings.NewBindings[bindings.OptimismPortal2](
				bindings.WithClient(elNode.EthClient()),
				bindings.WithTo(l1PortalAddr),
				bindings.WithTest(t))

			respectedGameType, err := contractio.Read(l1Portal.RespectedGameType(), t.Ctx())
			t.Require().NoError(err, "Failed to read respected game type")
			t.Gate().EqualValuesf(gameType, respectedGameType, "Respected game type must be %s", gameType)
		}
	})
}

func WithProposerGameType(gameType gameTypes.GameType) Option {
	return WithProposerOption(func(id sysgo.ComponentTarget, cfg *ps.CLIConfig) {
		cfg.DisputeGameType = uint32(gameType)
	})
}

func WithGuardianMatchL1PAO() Option {
	return WithDeployerOptions(
		sysgo.WithGuardianMatchL1PAO(),
	)
}

func WithFinalizationPeriodSeconds(n uint64) Option {
	return WithDeployerOptions(
		sysgo.WithFinalizationPeriodSeconds(n),
	)
}

func WithProofMaturityDelaySeconds(seconds uint64) Option {
	return WithDeployerOptions(
		sysgo.WithProofMaturityDelaySeconds(seconds),
	)
}

func WithDisputeGameFinalityDelaySeconds(seconds uint64) Option {
	return WithDeployerOptions(
		sysgo.WithDisputeGameFinalityDelaySeconds(seconds),
	)
}

// WithZK installs a shared ZK dispute game after the interop migration and
// starts the honest kona-sp1-proposer and op-challenger for it, sourcing super
// roots from the supernode. The SP1 super-aggregation vkey is loaded from
// KONA_SP1_ELF_DIR when the system starts. It also enables time travel and the
// timing defaults used by devstack ZK tests.
func WithZK() Option {
	return Combine(
		option{
			kinds: optionKindZKDisputeGame,
			applyFn: func(cfg *sysgo.PresetConfig) {
				cfg.ZKDisputeGame = &sysgo.ZKDisputeGameConfig{
					MaxChallengeDuration: DefaultZKChallengeDuration,
					MaxProveDuration:     DefaultZKProveDuration,
				}
			},
		},
		WithZKProposerOption(sysgo.WithZKProposalInterval(DefaultZKProposalInterval)),
		WithTimeTravelEnabled(),
		WithDisputeGameFinalityDelaySeconds(uint64(DefaultZKFinalityDelay/time.Second)),
		WithDeployerOptions(sysgo.WithJovianAtGenesis),
	)
}

// WithZKChallengeDuration overrides the maximum challenge duration configured
// by a preceding WithZK option.
func WithZKChallengeDuration(duration time.Duration) Option {
	return option{
		kinds: optionKindZKDisputeGame,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if cfg.ZKDisputeGame == nil {
				cfg.ZKDisputeGame = &sysgo.ZKDisputeGameConfig{}
			}
			cfg.ZKDisputeGame.MaxChallengeDuration = duration
		},
	}
}

// WithPreZKCutoverSuperGame seeds a valid SuperCannonKona game before WithZK
// retires that game type and installs the ZK dispute game.
func WithPreZKCutoverSuperGame() Option {
	return option{
		kinds: optionKindZKDisputeGame,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.SeedPreZKCutoverGame = true
		},
	}
}
