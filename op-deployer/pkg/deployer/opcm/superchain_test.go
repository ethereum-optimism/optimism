package opcm

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeploySuperchainScriptForge(t *testing.T) {
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	deploySuperchain := NewDeploySuperchainForgeCaller(forgeClient)
	output, recompiled, err := deploySuperchain(context.Background(), DeploySuperchainInput{
		Guardian:                  common.BigToAddress(big.NewInt(1)),
		SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
		Paused:                    true,
	})

	require.NoError(t, err)
	require.False(t, recompiled)
	require.NotNil(t, output)
}
