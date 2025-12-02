package chainconfig

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-program/chainconfig/test"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/stretchr/testify/require"
)

// TestGetCustomRollupConfig tests loading the custom rollup configs from test embed FS.
func TestGetCustomRollupConfig(t *testing.T) {
	config, err := rollupConfigByChainID(eth.ChainIDFromUInt64(901), test.TestCustomChainConfigFS)
	require.NoError(t, err)
	require.Equal(t, config.L1ChainID.Uint64(), uint64(900))
	require.Equal(t, config.L2ChainID.Uint64(), uint64(901))

	_, err = rollupConfigByChainID(eth.ChainIDFromUInt64(900), test.TestCustomChainConfigFS)
	require.Error(t, err)
}

func TestGetCustomRollupConfig_Missing(t *testing.T) {
	_, err := rollupConfigByChainID(eth.ChainIDFromUInt64(11111), test.TestCustomChainConfigFS)
	require.ErrorIs(t, err, ErrMissingChainConfig)
}

// TestGetCustomChainConfig tests loading the custom chain configs from test embed FS.
func TestGetCustomChainConfig(t *testing.T) {
	config, err := l2ChainConfigByChainID(eth.ChainIDFromUInt64(901), test.TestCustomChainConfigFS)
	require.NoError(t, err)
	require.Equal(t, config.ChainID.Uint64(), uint64(901))

	_, err = l2ChainConfigByChainID(eth.ChainIDFromUInt64(900), test.TestCustomChainConfigFS)
	require.Error(t, err)
}

func TestGetCustomChainConfig_Missing(t *testing.T) {
	_, err := l2ChainConfigByChainID(eth.ChainIDFromUInt64(11111), test.TestCustomChainConfigFS)
	require.ErrorIs(t, err, ErrMissingChainConfig)
}

func TestGetCustomL1ChainConfig(t *testing.T) {
	config, err := l1ChainConfigByChainID(eth.ChainIDFromUInt64(900), test.TestCustomChainConfigFS)
	require.NoError(t, err)
	require.Equal(t, config.ChainID.Uint64(), uint64(900))
}

func TestGetCustomL1ChainConfig_Missing(t *testing.T) {
	_, err := l1ChainConfigByChainID(eth.ChainIDFromUInt64(11111), test.TestCustomChainConfigFS)
	require.ErrorIs(t, err, ErrMissingChainConfig)
}

func TestGetCustomL1ChainConfig_KnownChainID(t *testing.T) {
	knownChainIds := []eth.ChainID{
		eth.ChainIDFromUInt64(1),        // Mainnet
		eth.ChainIDFromUInt64(11155111), // Sepolia
		eth.ChainIDFromUInt64(17000),    // Holesky
		eth.ChainIDFromUInt64(560048),   // Hoodi
	}
	for _, chainID := range knownChainIds {
		cfg, err := L1ChainConfigByChainID(chainID)
		require.NoError(t, err)
		require.True(t, chainID.Cmp(eth.ChainIDFromBig(cfg.ChainID)) == 0)
	}
	unknownChainId := eth.ChainIDFromUInt64(11111)
	_, err := L1ChainConfigByChainID(unknownChainId)
	require.ErrorIs(t, err, ErrMissingChainConfig)
}

func TestGetCustomDependencySetConfig(t *testing.T) {
	depSet, err := dependencySetByChainID(eth.ChainIDFromUInt64(901), test.TestCustomChainConfigFS)
	require.NoError(t, err)
	require.True(t, depSet.HasChain(eth.ChainIDFromUInt64(901)))
	require.True(t, depSet.HasChain(eth.ChainIDFromUInt64(902)))
	// Can use any chain ID from the dependency set
	depSet, err = dependencySetByChainID(eth.ChainIDFromUInt64(902), test.TestCustomChainConfigFS)
	require.NoError(t, err)
	require.True(t, depSet.HasChain(eth.ChainIDFromUInt64(901)))
	require.True(t, depSet.HasChain(eth.ChainIDFromUInt64(902)))

	_, err = dependencySetByChainID(eth.ChainIDFromUInt64(900), test.TestCustomChainConfigFS)
	require.Error(t, err)
}

func TestGetCustomDependencySetConfig_MissingConfig(t *testing.T) {
	_, err := dependencySetByChainID(eth.ChainIDFromUInt64(11111), test.TestCustomChainConfigEmptyFS)
	require.ErrorIs(t, err, ErrMissingChainConfig)
}

func TestListCustomChainIDs(t *testing.T) {
	actual, err := customChainIDs(test.TestCustomChainConfigFS)
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{eth.ChainIDFromUInt64(901)}, actual)
}

func TestLoadDependencySetFromRegistry(t *testing.T) {
	chainID, err := superchain.ChainIDByName("op-mainnet")
	require.NoError(t, err)
	depSet, err := DependencySetByChainID(eth.ChainIDFromUInt64(chainID))
	require.NoError(t, err)
	require.True(t, depSet.HasChain(eth.ChainIDFromUInt64(chainID)))
}

func TestCheckConfigFilenames(t *testing.T) {
	err := checkConfigFilenames(test.TestCustomChainConfigFS, "configs")
	require.NoError(t, err)
}

func TestCheckConfigFilenames_WithoutCustomL1Genesis(t *testing.T) {
	err := checkConfigFilenames(test.TestCustomChainConfigNoL1FS, "configs_no_l1")
	require.NoError(t, err)
}

func TestCheckConfigFilenames_MultipleL1Genesis(t *testing.T) {
	err := checkConfigFilenames(test.TestCustomChainConfigMultipleL1FS, "configs_multiple_l1")
	require.NoError(t, err)
}

func TestCheckConfigFilenames_Missing(t *testing.T) {
	err := checkConfigFilenames(test.TestCustomChainConfigEmptyFS, "configs_empty")
	require.NoError(t, err)
}

func TestCheckConfigFilenames_Invalid(t *testing.T) {
	err := checkConfigFilenames(test.TestCustomChainConfigTypoFS, "configs_typo")
	require.ErrorContains(t, err, "invalid config file name: genesis-l2-901.json")
}
