package opcm

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum/go-ethereum/common"
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
			"guardian":                  common.BigToAddress(big.NewInt(1)),
			"superchainProxyAdminOwner": common.BigToAddress(big.NewInt(3)),
			"paused":                    true,
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)
	})
}

func TestNewDeploySuperchainScriptForge(t *testing.T) {
	_, afacts := testutil.LocalArtifacts(t)
	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", afacts))
	require.NoError(t, err)

	deploySuperchain := NewDeploySuperchainForgeCaller(forgeClient)
	output, recompiled, err := deploySuperchain(context.Background(), DeploySuperchainInput{
		"guardian":                  common.BigToAddress(big.NewInt(1)),
		"superchainProxyAdminOwner": common.BigToAddress(big.NewInt(3)),
		"paused":                    true,
	})

	require.NoError(t, err)
	require.False(t, recompiled)
	require.NotNil(t, output)
}
