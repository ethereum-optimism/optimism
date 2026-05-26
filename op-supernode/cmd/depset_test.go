package main

import (
	"os"
	"path/filepath"
	"testing"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/stretchr/testify/require"
)

func TestApplySupernodeDependencySet_NoPath_LeavesVnCfgsUntouched(t *testing.T) {
	vnCfgs := map[eth.ChainID]*opnodecfg.Config{
		eth.ChainIDFromUInt64(10): {},
		eth.ChainIDFromUInt64(20): {},
	}
	require.NoError(t, applySupernodeDependencySet(&config.CLIConfig{}, vnCfgs))
	for _, vn := range vnCfgs {
		require.Nil(t, vn.DependencySet)
	}
}

func TestApplySupernodeDependencySet_LoadsAndAppliesToAllChains(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "depset.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"dependencies":{"4242":{}}}`), 0o600))

	vnCfgs := map[eth.ChainID]*opnodecfg.Config{
		eth.ChainIDFromUInt64(10): {},
		eth.ChainIDFromUInt64(20): {},
	}

	require.NoError(t, applySupernodeDependencySet(&config.CLIConfig{DependencySetPath: path}, vnCfgs))

	for chainID, vn := range vnCfgs {
		require.NotNilf(t, vn.DependencySet, "chain %s missing depset", chainID)
		require.True(t, vn.DependencySet.HasChain(eth.ChainIDFromUInt64(4242)),
			"depset for chain %s does not contain the loaded chain", chainID)
	}
}

func TestApplySupernodeDependencySet_FileMissing_ReturnsError(t *testing.T) {
	vnCfgs := map[eth.ChainID]*opnodecfg.Config{
		eth.ChainIDFromUInt64(10): {},
	}
	err := applySupernodeDependencySet(&config.CLIConfig{DependencySetPath: "/no/such/file.json"}, vnCfgs)
	require.Error(t, err)
}
