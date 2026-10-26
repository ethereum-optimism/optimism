package opcm

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDeploySuperchain(t *testing.T) {
	env := newTestGenesisEnv(t)
	output, err := DeploySuperchain(env, DeploySuperchainInput{
		Guardian:                  common.BigToAddress(big.NewInt(1)),
		SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
		Paused:                    true,
	})
	require.NoError(t, err)
	requireCode(t, env, output.SuperchainConfigImpl)
	requireCode(t, env, output.SuperchainConfigProxy)
	requireCode(t, env, output.SuperchainProxyAdmin)
}

// TestDeploySuperchain_EmbeddedBundle runs DeploySuperchain from the embedded contracts bundle.
func TestDeploySuperchain_EmbeddedBundle(t *testing.T) {
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient := testutil.NewForgeClient(t, fmt.Sprintf("%v", embeddedArtifactsFS))

	deploySuperchain := NewDeploySuperchainForgeCaller(forgeClient)
	output, err := deploySuperchain(context.Background(), DeploySuperchainInput{
		Guardian:                  common.BigToAddress(big.NewInt(1)),
		SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
		Paused:                    true,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
}
