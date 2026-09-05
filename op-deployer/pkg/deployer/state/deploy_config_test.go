package state

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// newCombineFixture returns a minimal intent/state fixture that passes
// CombineDeployConfig's validation.
func newCombineFixture() (Intent, ChainIntent, State, ChainState) {
	intent := Intent{
		L1ChainID:          1,
		L1ContractsLocator: artifacts.EmbeddedLocator,
	}
	chainState := ChainState{
		ID: common.HexToHash("0x123"),
	}
	chainIntent := ChainIntent{
		Eip1559Denominator:         1,
		Eip1559Elasticity:          2,
		GasLimit:                   standard.GasLimit,
		BaseFeeVaultRecipient:      common.HexToAddress("0x123"),
		L1FeeVaultRecipient:        common.HexToAddress("0x456"),
		SequencerFeeVaultRecipient: common.HexToAddress("0x789"),
		OperatorFeeVaultRecipient:  common.HexToAddress("0xabc"),
		Roles: ChainRoles{
			SystemConfigOwner: common.HexToAddress("0x123"),
			L1ProxyAdminOwner: common.HexToAddress("0x456"),
			L2ProxyAdminOwner: common.HexToAddress("0x789"),
			UnsafeBlockSigner: common.HexToAddress("0xabc"),
			Batcher:           common.HexToAddress("0xdef"),
		},
		// CustomGasToken defaults to disabled (all fields nil/empty)
		CustomGasToken: CustomGasToken{},
	}
	state := State{
		SuperchainDeployment: &addresses.SuperchainContracts{},
	}
	return intent, chainIntent, state, chainState
}

func TestCombineDeployConfig(t *testing.T) {
	intent, chainIntent, state, chainState := newCombineFixture()

	// apply hard fork overrides
	chainIntent.DeployOverrides = map[string]any{
		"l2GenesisFjordTimeOffset":    "0x1",
		"l2GenesisGraniteTimeOffset":  "0x2",
		"l2GenesisHoloceneTimeOffset": "0x3",
		"l2GenesisIsthmusTimeOffset":  "0x4",
		"l2GenesisJovianTimeOffset":   "0x5",
		"l2GenesisKarstTimeOffset":    "0x6",
		"l2GenesisLagoonTimeOffset":   "0x7",
	}

	out, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
	require.NoError(t, err)
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisFjordTimeOffset, hexutil.Uint64(1))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisGraniteTimeOffset, hexutil.Uint64(2))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisHoloceneTimeOffset, hexutil.Uint64(3))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisIsthmusTimeOffset, hexutil.Uint64(4))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisJovianTimeOffset, hexutil.Uint64(5))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisKarstTimeOffset, hexutil.Uint64(6))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisLagoonTimeOffset, hexutil.Uint64(7))
}

func TestCombineDeployConfig_RejectsAltDAOverrides(t *testing.T) {
	t.Run("global override", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		intent.GlobalDeployOverrides = map[string]any{"USEALTDA": false}

		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorIs(t, err, ErrAltDANoLongerSupported)
	})

	t.Run("detached chain override", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		intent.Chains = []*ChainIntent{{ID: common.HexToHash("0x456")}}
		chainIntent.DeployOverrides = map[string]any{"DaChAlLeNgEpRoXy": common.Address{}}

		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorIs(t, err, ErrAltDANoLongerSupported)
	})
}

func TestCombineDeployConfig_RejectsLegacyAltDAInputsWithoutMutation(t *testing.T) {
	t.Run("detached chain config", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		legacy := &legacyAltDAConfig{UseAltDA: true}
		chainIntent.LegacyDangerousAltDAConfig = legacy

		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorIs(t, err, ErrAltDANoLongerSupported)
		require.Same(t, legacy, chainIntent.LegacyDangerousAltDAConfig)
	})

	t.Run("detached chain state address", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		legacyAddress := common.HexToAddress("0x1234")
		legacyAddressPtr := &legacyAddress
		chainState.LegacyAltDAChallengeProxy = legacyAddressPtr

		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorIs(t, err, ErrAltDANoLongerSupported)
		require.Same(t, legacyAddressPtr, chainState.LegacyAltDAChallengeProxy)
	})

	t.Run("state snapshot config", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		legacy := &legacyAltDAConfig{DAChallengeWindow: 1}
		state.AppliedIntent = &Intent{Chains: []*ChainIntent{{LegacyDangerousAltDAConfig: legacy}}}

		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorIs(t, err, ErrAltDANoLongerSupported)
		require.Same(t, legacy, state.AppliedIntent.Chains[0].LegacyDangerousAltDAConfig)
	})
}

func TestCombineDeployConfig_GenesisTime(t *testing.T) {
	t.Run("unset leaves the deploy config timestamp nil", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		out, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.NoError(t, err)
		require.Nil(t, out.L2GenesisBlockTimestamp)
	})

	t.Run("committed genesis time in chain state propagates to the deploy config", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		genesisTime := hexutil.Uint64(1_750_000_000)
		chainState.GenesisTime = &genesisTime
		out, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.NoError(t, err)
		require.NotNil(t, out.L2GenesisBlockTimestamp)
		require.Equal(t, genesisTime, *out.L2GenesisBlockTimestamp)
	})
}

func TestCombineDeployConfig_PinnedOverrides(t *testing.T) {
	genesisTime := hexutil.Uint64(1_750_000_000)

	t.Run("a pinned genesis time rejects the timestamp override in chain's intent even when it matches", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		chainState.GenesisTime = &genesisTime
		chainIntent.DeployOverrides = map[string]any{"l2GenesisBlockTimestamp": genesisTime}
		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorContains(t, err, `deployOverrides key "l2GenesisBlockTimestamp" conflicts with the anchor commitment`)
	})

	t.Run("a pinned genesis time rejects the global starting block tag override", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		chainState.GenesisTime = &genesisTime
		intent.GlobalDeployOverrides = map[string]any{"l1StartingBlockTag": "0x1111111111111111111111111111111111111111111111111111111111111111"}
		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorContains(t, err, `globalDeployOverrides key "l1StartingBlockTag" conflicts with the anchor commitment`)
	})

	t.Run("reserved keys matches are case insensitive", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		chainState.GenesisTime = &genesisTime
		chainIntent.DeployOverrides = map[string]any{"L2GenesisBlockTimestamp": hexutil.Uint64(1)}
		_, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.ErrorContains(t, err, `deployOverrides key "L2GenesisBlockTimestamp" conflicts with the anchor commitment`)
	})

	t.Run("without a pin the timestamp override keeps its legacy behavior", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		override := hexutil.Uint64(1_760_000_000)
		chainIntent.DeployOverrides = map[string]any{"l2GenesisBlockTimestamp": override}
		out, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.NoError(t, err)
		require.NotNil(t, out.L2GenesisBlockTimestamp)
		require.Equal(t, override, *out.L2GenesisBlockTimestamp)
	})

	t.Run("a pinned genesis time still accepts unrelated overrides", func(t *testing.T) {
		intent, chainIntent, state, chainState := newCombineFixture()
		chainState.GenesisTime = &genesisTime
		chainIntent.DeployOverrides = map[string]any{"l2BlockTime": 1}
		out, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
		require.NoError(t, err)
		require.EqualValues(t, 1, out.L2BlockTime)
		require.Equal(t, genesisTime, *out.L2GenesisBlockTimestamp)
	})
}
