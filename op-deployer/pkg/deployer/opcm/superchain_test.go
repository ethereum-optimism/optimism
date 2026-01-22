package opcm

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestNewDeploySuperchainScript(t *testing.T) {
	// First we grab a test host
	host1 := createTestHost(t)

	// Then we load the script
	//
	// This would raise an error if the Go types didn't match the ABI
	deploySuperchain, err := NewDeploySuperchainScript(host1)
	require.NoError(t, err)

	deployInput := DeploySuperchainInput{
		Guardian:                   common.BigToAddress(big.NewInt(1)),
		ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
		SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
		Paused:                     true,
		RecommendedProtocolVersion: params.ProtocolVersion{1},
		RequiredProtocolVersion:    params.ProtocolVersion{2},
		IsOPCMv2:                   false,
	}
	t.Run("should succeed with OPCM v1 and protocol versions", func(t *testing.T) {
		// Then we deploy
		output, err := deploySuperchain.Run(deployInput)

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)
		require.NotEqual(t, common.Address{}, output.ProtocolVersionsProxy)
		require.NotEqual(t, common.Address{}, output.ProtocolVersionsImpl)
	})

	t.Run("should succeed with OPCM v2 and protocol versions deprecated", func(t *testing.T) {
		// Set isOPCMv2 and clear the protocol versions arguments
		deployInput.ProtocolVersionsOwner = common.Address{}
		deployInput.RecommendedProtocolVersion = params.ProtocolVersion{}
		deployInput.RequiredProtocolVersion = params.ProtocolVersion{}
		deployInput.IsOPCMv2 = true

		output, err := deploySuperchain.Run(deployInput)

		require.NoError(t, err)
		require.NotNil(t, output)
		require.Equal(t, common.Address{}, output.ProtocolVersionsProxy)
		require.Equal(t, common.Address{}, output.ProtocolVersionsImpl)
		require.NotEqual(t, common.Address{}, output.SuperchainConfigProxy)
		require.NotEqual(t, common.Address{}, output.SuperchainConfigImpl)
		require.NotEqual(t, common.Address{}, output.SuperchainProxyAdmin)
	})
}

func TestNewDeploySuperchainScriptForge(t *testing.T) {
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)
	deploySuperchain := NewDeploySuperchainForgeCaller(forgeClient)
	deployInput := DeploySuperchainInput{
		Guardian:                   common.BigToAddress(big.NewInt(1)),
		ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
		SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
		Paused:                     true,
		RecommendedProtocolVersion: params.ProtocolVersion{1},
		RequiredProtocolVersion:    params.ProtocolVersion{2},
		IsOPCMv2:                   false,
	}
	t.Run("should succeed with OPCM v1 and protocol versions", func(t *testing.T) {
		output, _, err := deploySuperchain(context.Background(), deployInput)

		require.NoError(t, err)
		require.NotNil(t, output)
		require.NotEqual(t, common.Address{}, output.ProtocolVersionsProxy)
		require.NotEqual(t, common.Address{}, output.ProtocolVersionsImpl)
	})

	t.Run("should succeed with OPCM v2 and protocol versions deprecated", func(t *testing.T) {
		deployInput.IsOPCMv2 = true
		deployInput.ProtocolVersionsOwner = common.Address{}
		deployInput.RecommendedProtocolVersion = params.ProtocolVersion{}
		deployInput.RequiredProtocolVersion = params.ProtocolVersion{}

		output, _, err := deploySuperchain(context.Background(), deployInput)

		require.NoError(t, err)
		require.NotNil(t, output)
		require.Equal(t, common.Address{}, output.ProtocolVersionsProxy)
		require.Equal(t, common.Address{}, output.ProtocolVersionsImpl)
		require.NotEqual(t, common.Address{}, output.SuperchainConfigProxy)
		require.NotEqual(t, common.Address{}, output.SuperchainConfigImpl)
		require.NotEqual(t, common.Address{}, output.SuperchainProxyAdmin)
	})
}
