package opcm

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestNewDeploySuperchainScript(t *testing.T) {
	t.Run("should not fail with current version of DeploySuperchain2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// Then we load the script
		//
		// This would raise an error if the Go types didn't match the ABI
		deploySuperchain, err := NewDeploySuperchainScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deploySuperchain.Run(DeploySuperchainInput{
			Guardian:                   common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
			SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
			Paused:                     true,
			RecommendedProtocolVersion: params.ProtocolVersion{1},
			RequiredProtocolVersion:    params.ProtocolVersion{2},
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)
	})
}

func TestNewDeploySuperchainScriptForge(t *testing.T) {
	tmpDir := t.TempDir()

	bundleDir, err := artifacts.ExtractEmbeddedForForge(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(bundleDir)
	require.NoError(t, err)
	t.Cleanup(func() { forgeClient.Close() })

	deploySuperchain := NewDeploySuperchainForgeCaller(forgeClient)
	output, recompiled, err := deploySuperchain(context.Background(), DeploySuperchainInput{
		Guardian:                   common.BigToAddress(big.NewInt(1)),
		ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
		SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
		Paused:                     true,
		RecommendedProtocolVersion: params.ProtocolVersion{1},
		RequiredProtocolVersion:    params.ProtocolVersion{2},
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	// The script should not be recompiled - forge should use the pre-warmed cache
	require.False(t, recompiled, "forge script should use pre-warmed cache and not recompile")
}
