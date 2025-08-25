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
	config, err := chainConfigByChainID(eth.ChainIDFromUInt64(901), test.TestCustomChainConfigFS)
	require.NoError(t, err)
	require.Equal(t, config.ChainID.Uint64(), uint64(901))

	_, err = chainConfigByChainID(eth.ChainIDFromUInt64(900), test.TestCustomChainConfigFS)
	require.Error(t, err)
}

func TestGetCustomChainConfig_Missing(t *testing.T) {
	_, err := chainConfigByChainID(eth.ChainIDFromUInt64(11111), test.TestCustomChainConfigFS)
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

func TestCheckConfigFilenames_Missing(t *testing.T) {
	err := checkConfigFilenames(test.TestCustomChainConfigEmptyFS, "configs_empty")
	require.NoError(t, err)
}

func TestCheckConfigFilenames_Invalid(t *testing.T) {
	err := checkConfigFilenames(test.TestCustomChainConfigTypoFS, "configs_typo")
	require.ErrorContains(t, err, "invalid config file name: genesis-l2-901.json")
}
