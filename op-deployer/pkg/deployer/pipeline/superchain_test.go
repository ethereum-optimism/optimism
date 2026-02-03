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
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestDeploySuperchain_WithForge(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	lgr := testlog.Logger(t, slog.LevelInfo)
	anvil, err := devnet.NewAnvil(lgr)
	require.NoError(t, err)
	require.NoError(t, anvil.Start())
	t.Cleanup(func() {
		require.NoError(t, anvil.Stop())
	})

	l1RPCUrl := anvil.RPCUrl()
	privateKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		lgr,
		common.Address{'D'},
		afacts,
	)
	require.NoError(t, err)

	opcmScripts, err := opcm.NewScripts(host)
	require.NoError(t, err)

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

	pEnv := &Env{
		Logger:      lgr,
		Scripts:     opcmScripts,
		ForgeClient: forgeClient,
		UseForge:    true,
		Context:     ctx,
		Broadcaster: broadcaster.NoopBroadcaster(),
		StateWriter: NoopStateWriter(),
		L1RPCUrl:    l1RPCUrl,
		PrivateKey:  privateKey,
	}

	err = DeploySuperchain(pEnv, intent, st)
	require.NoError(t, err)

	require.NotNil(t, st.SuperchainDeployment)
	require.NotNil(t, st.SuperchainRoles)

	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainProxyAdminImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl)

	require.Equal(t, intent.SuperchainRoles.SuperchainProxyAdminOwner, st.SuperchainRoles.SuperchainProxyAdminOwner)
	require.Equal(t, intent.SuperchainRoles.ProtocolVersionsOwner, st.SuperchainRoles.ProtocolVersionsOwner)
	require.Equal(t, intent.SuperchainRoles.SuperchainGuardian, st.SuperchainRoles.SuperchainGuardian)
}

func TestDeploySuperchain_WithForgeEverywhere(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	lgr := testlog.Logger(t, slog.LevelInfo)
	anvil, err := devnet.NewAnvil(lgr)
	require.NoError(t, err)
	require.NoError(t, anvil.Start())
	t.Cleanup(func() {
		require.NoError(t, anvil.Stop())
	})

	l1RPCUrl := anvil.RPCUrl()
	privateKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		lgr,
		common.Address{'D'},
		afacts,
	)
	require.NoError(t, err)

	opcmScripts, err := opcm.NewScripts(host)
	require.NoError(t, err)

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

	pEnv := &Env{
		Logger:      lgr,
		Scripts:     opcmScripts,
		ForgeClient: forgeClient,
		UseForge:    true,
		Context:     ctx,
		Broadcaster: broadcaster.NoopBroadcaster(),
		StateWriter: NoopStateWriter(),
		L1RPCUrl:    l1RPCUrl,
		PrivateKey:  privateKey,
	}

	err = DeploySuperchain(pEnv, intent, st)
	require.NoError(t, err)

	require.NotNil(t, st.SuperchainDeployment)
	require.NotNil(t, st.SuperchainRoles)

	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainProxyAdminImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl)
}

func TestDeploySuperchain_WithForge_ManualCall(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	deploySuperchain := opcm.NewDeploySuperchainForgeCaller(forgeClient)

	input := opcm.DeploySuperchainInput{
		Guardian:                   common.BigToAddress(big.NewInt(1)),
		ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
		SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
		Paused:                     false,
		RequiredProtocolVersion:    params.ProtocolVersion(rollup.OPStackSupport),
		RecommendedProtocolVersion: params.ProtocolVersion(rollup.OPStackSupport),
	}

	output, recompiled, err := deploySuperchain(ctx, input)
	require.NoError(t, err)
	require.False(t, recompiled, "script should not be recompiled")
	require.NotNil(t, output)

	require.NotEqual(t, common.Address{}, output.SuperchainProxyAdmin)
	require.NotEqual(t, common.Address{}, output.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, output.SuperchainConfigImpl)
	require.NotEqual(t, common.Address{}, output.ProtocolVersionsProxy)
	require.NotEqual(t, common.Address{}, output.ProtocolVersionsImpl)
}
