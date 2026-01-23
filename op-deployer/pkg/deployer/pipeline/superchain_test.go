package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestDeploySuperchain_WithForge(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Extract embedded artifacts
	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	// Create Forge client
	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	// Create a test host for other scripts (even though we won't use it for DeploySuperchain)
	// We use LocalArtifacts which should have compatible versions
	_, afacts := testutil.LocalArtifacts(t)
	lgr := testlog.Logger(t, slog.LevelInfo)
	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		lgr,
		common.Address{'D'},
		afacts,
	)
	require.NoError(t, err)

	// Load scripts (needed for Env, even though we'll use Forge for DeploySuperchain)
	opcmScripts, err := opcm.NewScripts(host)
	require.NoError(t, err)

	// Create test input
	intent := &state.Intent{
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:     common.BigToAddress(big.NewInt(2)),
			SuperchainGuardian:        common.BigToAddress(big.NewInt(3)),
		},
	}
	st := &state.State{
		Version: 1,
	}

	// Create Env with Forge enabled
	pEnv := &Env{
		Logger:      lgr,
		Scripts:     opcmScripts,
		ForgeClient: forgeClient,
		UseForge:    true,
		Context:     ctx,
		Broadcaster: broadcaster.NoopBroadcaster(),
		StateWriter: NoopStateWriter(),
	}

	// Test DeploySuperchain with Forge
	err = DeploySuperchain(pEnv, intent, st)
	require.NoError(t, err)

	// Verify the deployment was successful
	require.NotNil(t, st.SuperchainDeployment)
	require.NotNil(t, st.SuperchainRoles)

	// Verify addresses are set
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainProxyAdminImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl)

	// Verify roles match
	require.Equal(t, intent.SuperchainRoles.SuperchainProxyAdminOwner, st.SuperchainRoles.SuperchainProxyAdminOwner)
	require.Equal(t, intent.SuperchainRoles.ProtocolVersionsOwner, st.SuperchainRoles.ProtocolVersionsOwner)
	require.Equal(t, intent.SuperchainRoles.SuperchainGuardian, st.SuperchainRoles.SuperchainGuardian)
}

func TestDeploySuperchain_WithForgeEverywhere(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Extract embedded artifacts
	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	// Create Forge client
	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	// Create a test host for other scripts (even though we won't use it for DeploySuperchain)
	_, afacts := testutil.LocalArtifacts(t)
	lgr := testlog.Logger(t, slog.LevelInfo)
	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		lgr,
		common.Address{'D'},
		afacts,
	)
	require.NoError(t, err)

	// Load scripts (needed for Env, even though we'll use Forge for DeploySuperchain)
	opcmScripts, err := opcm.NewScripts(host)
	require.NoError(t, err)

	// Create test input
	intent := &state.Intent{
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:     common.BigToAddress(big.NewInt(2)),
			SuperchainGuardian:        common.BigToAddress(big.NewInt(3)),
		},
	}
	st := &state.State{
		Version: 1,
	}

	// Create Env with UseForge enabled
	pEnv := &Env{
		Logger:      lgr,
		Scripts:     opcmScripts,
		ForgeClient: forgeClient,
		UseForge:    true,
		Context:     ctx,
		Broadcaster: broadcaster.NoopBroadcaster(),
		StateWriter: NoopStateWriter(),
	}

	// Test DeploySuperchain with Forge
	err = DeploySuperchain(pEnv, intent, st)
	require.NoError(t, err)

	// Verify the deployment was successful
	require.NotNil(t, st.SuperchainDeployment)
	require.NotNil(t, st.SuperchainRoles)

	// Verify addresses are set
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainProxyAdminImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl)
}

func TestDeploySuperchain_WithForge_ManualCall(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Extract embedded artifacts
	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	// Create Forge client
	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	// Create Forge caller directly (similar to TestNewDeploySuperchainScriptForge)
	deploySuperchain := opcm.NewDeploySuperchainForgeCaller(forgeClient)

	// Create input matching what DeploySuperchain would use
	input := opcm.DeploySuperchainInput{
		Guardian:                   common.BigToAddress(big.NewInt(1)),
		ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
		SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
		Paused:                     false,
		RequiredProtocolVersion:    params.ProtocolVersion(rollup.OPStackSupport),
		RecommendedProtocolVersion: params.ProtocolVersion(rollup.OPStackSupport),
	}

	// Call Forge script
	output, recompiled, err := deploySuperchain(ctx, input)
	require.NoError(t, err)
	require.False(t, recompiled, "script should not be recompiled")
	require.NotNil(t, output)

	// Verify output addresses are set
	require.NotEqual(t, common.Address{}, output.SuperchainProxyAdmin)
	require.NotEqual(t, common.Address{}, output.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, output.SuperchainConfigImpl)
	require.NotEqual(t, common.Address{}, output.ProtocolVersionsProxy)
	require.NotEqual(t, common.Address{}, output.ProtocolVersionsImpl)
}
