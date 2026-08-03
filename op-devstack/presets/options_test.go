package presets

import (
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestOptionKindsFromCompositeOptions(t *testing.T) {
	t.Run("WithSequencingWindow", func(t *testing.T) {
		require.Equal(t,
			optionKindDeployer|optionKindMaxSequencingWindow,
			WithSequencingWindow(12, 24).optionKinds(),
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
			RequireGameTypePresent(gameTypes.CannonKonaGameType).optionKinds(),
		)
	})

	t.Run("nil adapters do not claim support kinds", func(t *testing.T) {
		require.Zero(t, WithDeployerOptions(nil).optionKinds())
		require.Zero(t, WithLocalContractSourcesAt("").optionKinds())
		require.Zero(t, WithBatcherOption(nil).optionKinds())
		require.Zero(t, WithGlobalL2CLOption(nil).optionKinds())
		require.Zero(t, WithGlobalSyncTesterELOption(nil).optionKinds())
		require.Zero(t, WithProposerOption(nil).optionKinds())
		require.Zero(t, WithZKProposerOption(nil).optionKinds())
		require.Zero(t, WithOPRBuilderOption(nil).optionKinds())
		require.Zero(t, WithPreGenesisSuperGame().optionKinds())
		require.Zero(t, AfterBuild(nil).optionKinds())
	})
}

func TestWithLocalContractSourcesAt(t *testing.T) {
	cfg, _ := collectPresetConfig([]Option{WithLocalContractSourcesAt("/tmp/contracts-bedrock")})
	require.Equal(t, "/tmp/contracts-bedrock", cfg.LocalContractArtifactsPath)
}

func TestWithZK(t *testing.T) {
	want := sysgo.ZKDisputeGameConfig{
		MaxChallengeDuration: 30 * time.Minute,
		MaxProveDuration:     30 * time.Minute,
	}
	cfg, combined := collectPresetConfig([]Option{WithZK()})
	require.Equal(t,
		optionKindDeployer|optionKindTimeTravel|optionKindZKDisputeGame|optionKindZKProposer,
		combined.optionKinds(),
	)
	require.Equal(t, &want, cfg.ZKDisputeGame)
	require.Len(t, cfg.ZKProposerOptions, 1)
	require.True(t, cfg.EnableTimeTravel)
	require.Len(t, cfg.DeployerOptions, 2)
}

func TestWithZKChallengeDuration(t *testing.T) {
	cfg, _ := collectPresetConfig([]Option{
		WithZK(),
		WithZKChallengeDuration(5 * time.Minute),
	})
	require.Equal(t, 5*time.Minute, cfg.ZKDisputeGame.MaxChallengeDuration)
}

func TestWithZKProposerOption(t *testing.T) {
	opt := sysgo.WithZKProposalInterval(12 * time.Second)
	cfg, combined := collectPresetConfig([]Option{WithZKProposerOption(opt)})
	require.Equal(t, optionKindZKProposer, combined.optionKinds())
	require.Len(t, cfg.ZKProposerOptions, 1)
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
				RequireGameTypePresent(gameTypes.CannonKonaGameType),
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
			name:      "minimal allows op-reth options",
			supported: minimalPresetSupportedOptionKinds,
			opts:      WithOpRethOption(sysgo.OpRethWithBinary("op-reth-superset")),
			want:      0,
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
			name:      "shared supernode proofs reject pre-genesis super game",
			supported: supernodeProofsPresetSupportedOptionKinds,
			opts: Combine(
				WithTimeTravelEnabled(),
				WithPreGenesisSuperGame(eth.Bytes32{0x01}, eth.Bytes32{0x02}),
			),
			want: optionKindPreGenesisSuperGame,
		},
		{
			name:      "two l2 supernode proofs accept pre-genesis super game",
			supported: twoL2SupernodeProofsPresetSupportedOptionKinds,
			opts: Combine(
				WithTimeTravelEnabled(),
				WithPreGenesisSuperGame(eth.Bytes32{0x01}, eth.Bytes32{0x02}),
			),
			want: 0,
		},
		{
			name:      "two l2 supernode proofs accept ZK",
			supported: twoL2SupernodeProofsPresetSupportedOptionKinds,
			opts:      WithZK(),
			want:      0,
		},
		{
			name:      "two l2 supernode proofs accept ZK proposer options",
			supported: twoL2SupernodeProofsPresetSupportedOptionKinds,
			opts:      WithZKProposerOption(sysgo.WithZKProposalInterval(time.Minute)),
			want:      0,
		},
		{
			name:      "single chain supernode proofs reject ZK",
			supported: supernodeProofsPresetSupportedOptionKinds,
			opts:      WithZK(),
			want:      optionKindZKDisputeGame | optionKindZKProposer,
		},
		{
			name:      "single chain supernode proofs reject ZK proposer options",
			supported: supernodeProofsPresetSupportedOptionKinds,
			opts:      WithZKProposerOption(sysgo.WithZKProposalInterval(time.Minute)),
			want:      optionKindZKProposer,
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
			opts: Combine(
				WithTimeTravelEnabled(),
				WithPreGenesisSuperGame(eth.Bytes32{0x01}, eth.Bytes32{0x02}),
			),
			want: 0,
		},
		{
			name:      "unsupported proof validation is called out separately from generic after build",
			supported: optionKindAfterBuild,
			opts:      RequireGameTypePresent(gameTypes.CannonKonaGameType),
			want:      optionKindProofValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, unsupportedPresetOptionKinds(tt.opts, tt.supported))
		})
	}
}

func TestValidatePresetConfigRejectsZKProposerOptionWithoutZK(t *testing.T) {
	cfg, _ := collectPresetConfig([]Option{
		WithZKProposerOption(sysgo.WithZKProposalInterval(time.Minute)),
	})
	require.EqualError(t, validatePresetConfig(cfg), "ZK proposer options require WithZK")
}
