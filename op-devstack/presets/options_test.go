package presets

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/stretchr/testify/require"
)

func TestOptionKindsFromCompositeOptions(t *testing.T) {
	t.Run("WithSequencingWindow", func(t *testing.T) {
		require.Equal(t,
			optionKindDeployer|optionKindMaxSequencingWindow,
			WithSequencingWindow(12, 24).optionKinds(),
		)
	})

	t.Run("WithCannonKonaGameTypeAdded", func(t *testing.T) {
		require.Equal(t,
			optionKindAddedGameType|optionKindChallengerCannonKona,
			WithCannonKonaGameTypeAdded().optionKinds(),
		)
	})

	t.Run("WithL1Geth", func(t *testing.T) {
		require.Equal(t,
			optionKindL1EL,
			WithL1Geth("/tmp/geth").optionKinds(),
		)
	})

	t.Run("RequireGameTypePresent", func(t *testing.T) {
		require.Equal(t,
			optionKindAfterBuild|optionKindProofValidation,
			RequireGameTypePresent(gameTypes.CannonGameType).optionKinds(),
		)
	})

	t.Run("nil adapters do not claim support kinds", func(t *testing.T) {
		require.Zero(t, WithDeployerOptions(nil).optionKinds())
		require.Zero(t, WithLocalContractSourcesAt("").optionKinds())
		require.Zero(t, WithBatcherOption(nil).optionKinds())
		require.Zero(t, WithGlobalL2CLOption(nil).optionKinds())
		require.Zero(t, WithGlobalSyncTesterELOption(nil).optionKinds())
		require.Zero(t, WithProposerOption(nil).optionKinds())
		require.Zero(t, WithOPRBuilderOption(nil).optionKinds())
		require.Zero(t, AfterBuild(nil).optionKinds())
	})
}

func TestWithLocalContractSourcesAt(t *testing.T) {
	cfg, _ := collectPresetConfig([]Option{WithLocalContractSourcesAt("/tmp/contracts-bedrock")})
	require.Equal(t, "/tmp/contracts-bedrock", cfg.LocalContractArtifactsPath)
}

func TestUnsupportedPresetOptionKinds(t *testing.T) {
	builderOpt := sysgo.OPRBuilderNodeOptionFn(func(devtest.CommonT, sysgo.ComponentTarget, *sysgo.OPRBuilderNodeConfig) {})

	tests := []struct {
		name      string
		supported optionKinds
		opts      Option
		want      optionKinds
	}{
		{
			name:      "minimal allows proof validation hooks",
			supported: minimalPresetSupportedOptionKinds,
			opts: Combine(
				WithTimeTravelEnabled(),
				RequireGameTypePresent(gameTypes.CannonGameType),
			),
			want: 0,
		},
		{
			name:      "minimal allows l1 EL override",
			supported: minimalPresetSupportedOptionKinds,
			opts:      WithL1Geth("/tmp/geth"),
			want:      0,
		},
		{
			name:      "minimal with conductors rejects challenger toggle",
			supported: minimalWithConductorsPresetSupportedOptionKinds,
			opts:      WithChallengerCannonKonaEnabled(),
			want:      optionKindChallengerCannonKona,
		},
		{
			name:      "flashblocks allows builder and deployer adapters",
			supported: singleChainWithFlashblocksPresetSupportedOptionKinds,
			opts: Combine(
				WithLocalContractSourcesAt("/tmp/contracts-bedrock"),
				WithOPRBuilderOption(builderOpt),
				WithTimeTravelEnabled(),
			),
			want: optionKindTimeTravel,
		},
		{
			name:      "simple interop super proofs reject builder and proof hooks",
			supported: simpleInteropSuperProofsPresetSupportedOptionKinds,
			opts: Combine(
				WithOPRBuilderOption(builderOpt),
				RequireGameTypePresent(gameTypes.CannonGameType),
			),
			want: optionKindOPRBuilder | optionKindAfterBuild | optionKindProofValidation,
		},
		{
			name:      "supernode proofs only allow challenger toggle",
			supported: supernodeProofsPresetSupportedOptionKinds,
			opts: Combine(
				WithChallengerCannonKonaEnabled(),
				WithTimeTravelEnabled(),
			),
			want: optionKindTimeTravel,
		},
		{
			name:      "two l2 supernode rejects time travel",
			supported: twoL2SupernodePresetSupportedOptionKinds,
			opts:      WithTimeTravelEnabled(),
			want:      optionKindTimeTravel,
		},
		{
			name:      "two l2 supernode interop accepts time travel",
			supported: twoL2SupernodeInteropPresetSupportedOptionKinds,
			opts:      WithTimeTravelEnabled(),
			want:      0,
		},
		{
			name:      "unsupported proof validation is called out separately from generic after build",
			supported: optionKindAfterBuild,
			opts:      RequireGameTypePresent(gameTypes.CannonGameType),
			want:      optionKindProofValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, unsupportedPresetOptionKinds(tt.opts, tt.supported))
		})
	}
}
