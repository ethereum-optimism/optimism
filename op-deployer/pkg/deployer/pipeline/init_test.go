package pipeline

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestInitLiveStrategy_OPCMReuseLogicSepolia(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	l1ChainID := uint64(11155111)
	t.Run("untagged L1 locator", func(t *testing.T) {
		st := &state.State{
			Version: 1,
		}
		require.NoError(t, InitLiveStrategy(
			ctx,
			&Env{
				L1Client: client,
				Logger:   lgr,
			},
			&state.Intent{
				L1ChainID:          l1ChainID,
				L1ContractsLocator: artifacts.MustNewLocatorFromURL("file:///not-a-path"),
				L2ContractsLocator: artifacts.MustNewLocatorFromURL("file:///not-a-path"),
			},
			st,
		))

		// Defining a file locator will always deploy a new superchain and OPCM
		require.Nil(t, st.SuperchainDeployment)
		require.Nil(t, st.ImplementationsDeployment)
	})

	t.Run("embedded L1 locator with standard intent types and standard roles", func(t *testing.T) {
		runTest := func(configType state.IntentType) {
			_, afacts := testutil.LocalArtifacts(t)
			host, err := env.DefaultForkedScriptHost(
				ctx,
				broadcaster.NoopBroadcaster(),
				testlog.Logger(t, log.LevelInfo),
				common.Address{'D'},
				afacts,
				rpcClient,
			)
			require.NoError(t, err)

			stdSuperchainRoles, err := state.GetStandardSuperchainRoles(l1ChainID)
			require.NoError(t, err)

			opcmAddr, err := standard.OPCMImplAddressFor(l1ChainID, standard.CurrentTag)
			require.NoError(t, err)

			intent := &state.Intent{
				ConfigType:         configType,
				L1ChainID:          l1ChainID,
				L1ContractsLocator: artifacts.EmbeddedLocator,
				L2ContractsLocator: artifacts.EmbeddedLocator,
				OPCMAddress:        &opcmAddr,
			}
			st := &state.State{
				Version: 1,
			}
			require.NoError(t, InitLiveStrategy(
				ctx,
				&Env{
					L1Client:     client,
					Logger:       lgr,
					L1ScriptHost: host,
				},
				intent,
				st,
			))

			// Defining a file locator will always deploy a new superchain and OPCM
			superCfg, err := standard.SuperchainFor(l1ChainID)
			require.NoError(t, err)
			proxyAdmin, err := standard.SuperchainProxyAdminAddrFor(l1ChainID)
			require.NoError(t, err)

			expDeployment := &addresses.SuperchainContracts{
				SuperchainProxyAdminImpl: proxyAdmin,
				ProtocolVersionsProxy:    superCfg.ProtocolVersionsAddr,
				ProtocolVersionsImpl:     common.HexToAddress("0x37E15e4d6DFFa9e5E320Ee1eC036922E563CB76C"),
				SuperchainConfigProxy:    superCfg.SuperchainConfigAddr,
				SuperchainConfigImpl:     common.HexToAddress("0xb08Cc720F511062537ca78BdB0AE691F04F5a957"),
			}

			// Tagged locator will reuse the existing superchain and OPCM
			require.NotNil(t, st.SuperchainDeployment)
			require.NotNil(t, st.ImplementationsDeployment)
			require.NotNil(t, st.SuperchainRoles)
			require.Equal(t, *expDeployment, *st.SuperchainDeployment)
			require.Equal(t, opcmAddr, st.ImplementationsDeployment.OpcmImpl)
			require.Equal(t, *stdSuperchainRoles, *st.SuperchainRoles)
		}

		runTest(state.IntentTypeStandard)
		runTest(state.IntentTypeStandardOverrides)
	})

	t.Run("tagged L1 locator with standard intent types and modified roles", func(t *testing.T) {
		runTest := func(configType state.IntentType) {
			intent := &state.Intent{
				ConfigType:         configType,
				L1ChainID:          l1ChainID,
				L1ContractsLocator: artifacts.DefaultL1ContractsLocator,
				L2ContractsLocator: artifacts.DefaultL2ContractsLocator,
				SuperchainRoles: &addresses.SuperchainRoles{
					SuperchainGuardian: common.Address{0: 99},
				},
			}
			st := &state.State{
				Version: 1,
			}
			require.NoError(t, InitLiveStrategy(
				ctx,
				&Env{
					L1Client: client,
					Logger:   lgr,
				},
				intent,
				st,
			))

			// Modified roles will cause a new superchain and OPCM to be deployed
			require.Nil(t, st.SuperchainDeployment)
			require.Nil(t, st.ImplementationsDeployment)
		}

		runTest(state.IntentTypeStandard)
		runTest(state.IntentTypeStandardOverrides)
	})

	t.Run("tagged locator with custom intent type", func(t *testing.T) {
		intent := &state.Intent{
			ConfigType:         state.IntentTypeCustom,
			L1ChainID:          l1ChainID,
			L1ContractsLocator: artifacts.DefaultL1ContractsLocator,
			L2ContractsLocator: artifacts.DefaultL2ContractsLocator,
			SuperchainRoles: &addresses.SuperchainRoles{
				SuperchainGuardian: common.Address{0: 99},
			},
		}
		st := &state.State{
			Version: 1,
		}
		require.NoError(t, InitLiveStrategy(
			ctx,
			&Env{
				L1Client: client,
				Logger:   lgr,
			},
			intent,
			st,
		))

		// Custom intent types always deploy a new superchain and OPCM
		require.Nil(t, st.SuperchainDeployment)
		require.Nil(t, st.ImplementationsDeployment)
	})
}

// TestPopulateSuperchainState validates that the ReadSuperchainDeployment script successfully returns data about the
// given Superchain. For testing purposes, we use a forked script host that points to a pinned block on Sepolia. Pinning
// the block lets us use constant values in the test without worrying about changes on chain. We use values from the SR
// whenever possible, however some (like the Superchain PAO) are not included in the SR and are therefore hardcoded.
func TestPopulateSuperchainState(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	host, err := env.ForkedScriptHost(
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		afacts,
		rpcClient,
		// corresponds to the latest block on sepolia as of 04/30/2025. used to prevent config drift on sepolia
		// from failing this test
		big.NewInt(8227159),
	)
	require.NoError(t, err)

	l1Versions, err := standard.L1VersionsFor(11155111)
	require.NoError(t, err)
	superchain, err := standard.SuperchainFor(11155111)
	require.NoError(t, err)
	opcmAddr := l1Versions["op-contracts/v2.0.0-rc.1"].OPContractsManager.Address

	t.Run("valid OPCM address only", func(t *testing.T) {
		dep, roles, err := PopulateSuperchainState(host, common.Address(*opcmAddr), common.Address{})
		require.NoError(t, err)
		require.Equal(t, addresses.SuperchainContracts{
			SuperchainProxyAdminImpl: common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"),
			SuperchainConfigProxy:    superchain.SuperchainConfigAddr,
			SuperchainConfigImpl:     common.HexToAddress("0x4da82a327773965b8d4D85Fa3dB8249b387458E7"),
			ProtocolVersionsProxy:    superchain.ProtocolVersionsAddr,
			ProtocolVersionsImpl:     common.HexToAddress("0x37E15e4d6DFFa9e5E320Ee1eC036922E563CB76C"),
		}, *dep)
		require.Equal(t, addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2"),
			ProtocolVersionsOwner:     common.HexToAddress("0xfd1D2e729aE8eEe2E146c033bf4400fE75284301"),
			SuperchainGuardian:        common.HexToAddress("0x7a50f00e8D05b95F98fE38d8BeE366a7324dCf7E"),
		}, *roles)
	})

	t.Run("OPCM address with SuperchainConfigProxy", func(t *testing.T) {
		// When both are provided and OPCM version < 7.0.0, the script uses v1 flow
		// The SuperchainConfigProxy parameter is ignored in v1 flow
		dep, roles, err := PopulateSuperchainState(host, common.Address(*opcmAddr), superchain.SuperchainConfigAddr)
		require.NoError(t, err)
		require.NotNil(t, dep)
		require.NotNil(t, roles)

		// For OPCMv1, ProtocolVersions should be populated (read from OPCM)
		require.NotEqual(t, common.Address{}, dep.ProtocolVersionsProxy, "ProtocolVersionsProxy should be populated for v1")
		require.NotEqual(t, common.Address{}, dep.ProtocolVersionsImpl, "ProtocolVersionsImpl should be populated for v1")
		require.NotEqual(t, common.Address{}, roles.ProtocolVersionsOwner, "ProtocolVersionsOwner should be populated for v1")

		// Verify that values match what OPCM returns (not the SuperchainConfigProxy parameter)
		require.Equal(t, superchain.SuperchainConfigAddr, dep.SuperchainConfigProxy)
		require.Equal(t, superchain.ProtocolVersionsAddr, dep.ProtocolVersionsProxy)
	})

	t.Run("invalid OPCM address", func(t *testing.T) {
		// Use an invalid address (non-existent contract)
		invalidOpcmAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
		dep, roles, err := PopulateSuperchainState(host, invalidOpcmAddr, common.Address{})
		require.Error(t, err)
		require.Nil(t, dep)
		require.Nil(t, roles)
		require.Contains(t, err.Error(), "error reading superchain deployment")
	})

	t.Run("output mapping validation", func(t *testing.T) {
		dep, roles, err := PopulateSuperchainState(host, common.Address(*opcmAddr), common.Address{})
		require.NoError(t, err)
		require.NotNil(t, dep)
		require.NotNil(t, roles)

		// Verify all SuperchainContracts fields are populated correctly
		require.NotEqual(t, common.Address{}, dep.SuperchainProxyAdminImpl, "SuperchainProxyAdminImpl should be populated")
		require.NotEqual(t, common.Address{}, dep.SuperchainConfigProxy, "SuperchainConfigProxy should be populated")
		require.NotEqual(t, common.Address{}, dep.SuperchainConfigImpl, "SuperchainConfigImpl should be populated")
		require.NotEqual(t, common.Address{}, dep.ProtocolVersionsProxy, "ProtocolVersionsProxy should be populated for v1")
		require.NotEqual(t, common.Address{}, dep.ProtocolVersionsImpl, "ProtocolVersionsImpl should be populated for v1")

		// Verify implementations are different from proxies
		require.NotEqual(t, dep.SuperchainConfigImpl, dep.SuperchainConfigProxy, "SuperchainConfigImpl should differ from proxy")
		require.NotEqual(t, dep.ProtocolVersionsImpl, dep.ProtocolVersionsProxy, "ProtocolVersionsImpl should differ from proxy")

		// Verify all SuperchainRoles fields are populated correctly
		require.NotEqual(t, common.Address{}, roles.SuperchainProxyAdminOwner, "SuperchainProxyAdminOwner should be populated")
		require.NotEqual(t, common.Address{}, roles.ProtocolVersionsOwner, "ProtocolVersionsOwner should be populated for v1")
		require.NotEqual(t, common.Address{}, roles.SuperchainGuardian, "SuperchainGuardian should be populated")

		// Verify expected values match
		require.Equal(t, superchain.SuperchainConfigAddr, dep.SuperchainConfigProxy)
		require.Equal(t, superchain.ProtocolVersionsAddr, dep.ProtocolVersionsProxy)
	})
}

// TestPopulateSuperchainState_OPCMV2 validates that PopulateSuperchainState handles the OPCM v2 flow, where only a SuperchainConfigProxy
// is provided. This test uses a forked script host configured to a pinned Sepolia block to guarantee deterministic results.
// It asserts that returned roles and addresses are correct for the superchain config under OPCM v2, and that ProtocolVersions
// contract fields—which are not present in OPCM v2—are zeroed out as expected.
func TestPopulateSuperchainState_OPCMV2(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	host, err := env.ForkedScriptHost(
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		afacts,
		rpcClient,
		// corresponds to the latest block on sepolia as of 04/30/2025. used to prevent config drift on sepolia
		// from failing this test
		big.NewInt(8227159),
	)
	require.NoError(t, err)

	superchain, err := standard.SuperchainFor(11155111)
	require.NoError(t, err)

	t.Run("SuperchainConfigProxy only", func(t *testing.T) {
		// opcmAddr is set to 0, all config is provided in the superchainConfigProxy
		dep, roles, err := PopulateSuperchainState(host, common.Address{}, superchain.SuperchainConfigAddr)
		require.NoError(t, err)

		require.Equal(t, addresses.SuperchainContracts{
			SuperchainProxyAdminImpl: common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"),
			SuperchainConfigProxy:    superchain.SuperchainConfigAddr,
			SuperchainConfigImpl:     common.HexToAddress("0x4da82a327773965b8d4D85Fa3dB8249b387458E7"),
			// TODO(#18612): Remove ProtocolVersions fields when OPCMv1 gets deprecated
			ProtocolVersionsProxy: common.Address{},
			ProtocolVersionsImpl:  common.Address{},
		}, *dep)
		require.Equal(t, addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2"),
			// TODO(#18612): Remove ProtocolVersions fields when OPCMv1 gets deprecated
			ProtocolVersionsOwner: common.Address{},
			SuperchainGuardian:    common.HexToAddress("0x7a50f00e8D05b95F98fE38d8BeE366a7324dCf7E"),
		}, *roles)
	})

	t.Run("both addresses zero", func(t *testing.T) {
		// When both are zero, the script detects OPCMv2 flow (because opcmAddr == 0)
		// but then requires SuperchainConfigProxy to be set, so it should error
		dep, roles, err := PopulateSuperchainState(host, common.Address{}, common.Address{})
		require.Error(t, err)
		require.Nil(t, dep)
		require.Nil(t, roles)
		require.Contains(t, err.Error(), "superchainConfigProxy required for OPCM v2")
	})

	t.Run("invalid SuperchainConfigProxy", func(t *testing.T) {
		// Use an invalid address (non-existent contract)
		invalidSuperchainConfigProxy := common.HexToAddress("0x1234567890123456789012345678901234567890")
		dep, roles, err := PopulateSuperchainState(host, common.Address{}, invalidSuperchainConfigProxy)
		require.Error(t, err)
		require.Nil(t, dep)
		require.Nil(t, roles)
		require.Contains(t, err.Error(), "error reading superchain deployment")
	})

	t.Run("output mapping validation", func(t *testing.T) {
		dep, roles, err := PopulateSuperchainState(host, common.Address{}, superchain.SuperchainConfigAddr)
		require.NoError(t, err)
		require.NotNil(t, dep)
		require.NotNil(t, roles)

		// Verify SuperchainConfig fields are populated
		require.NotEqual(t, common.Address{}, dep.SuperchainProxyAdminImpl, "SuperchainProxyAdminImpl should be populated")
		require.NotEqual(t, common.Address{}, dep.SuperchainConfigProxy, "SuperchainConfigProxy should be populated")
		require.NotEqual(t, common.Address{}, dep.SuperchainConfigImpl, "SuperchainConfigImpl should be populated")
		require.NotEqual(t, dep.SuperchainConfigImpl, dep.SuperchainConfigProxy, "SuperchainConfigImpl should differ from proxy")

		// Verify ProtocolVersions fields are zeroed for v2
		require.Equal(t, common.Address{}, dep.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for v2")
		require.Equal(t, common.Address{}, dep.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for v2")

		// Verify SuperchainRoles fields are populated correctly
		require.NotEqual(t, common.Address{}, roles.SuperchainProxyAdminOwner, "SuperchainProxyAdminOwner should be populated")
		require.NotEqual(t, common.Address{}, roles.SuperchainGuardian, "SuperchainGuardian should be populated")

		// Verify ProtocolVersionsOwner is zeroed for v2
		require.Equal(t, common.Address{}, roles.ProtocolVersionsOwner, "ProtocolVersionsOwner should be zero for v2")

		// Verify expected values match
		require.Equal(t, superchain.SuperchainConfigAddr, dep.SuperchainConfigProxy)
	})
}

// Validates the OPCM v2 flow in InitLiveStrategy
// when SuperchainConfigProxy is provided and opcmV2Enabled is true.
func TestInitLiveStrategy_OPCMV2WithSuperchainConfigProxy(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	l1ChainID := uint64(11155111)
	superchain, err := standard.SuperchainFor(l1ChainID)
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		afacts,
		rpcClient,
	)
	require.NoError(t, err)

	// Set opcmV2Enabled flag via devFeatureBitmap
	opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
	intent := &state.Intent{
		ConfigType:            state.IntentTypeStandard,
		L1ChainID:             l1ChainID,
		L1ContractsLocator:    artifacts.EmbeddedLocator,
		L2ContractsLocator:    artifacts.EmbeddedLocator,
		SuperchainConfigProxy: &superchain.SuperchainConfigAddr,
		GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": opcmV2Flag,
		},
	}

	st := &state.State{
		Version: 1,
	}

	err = InitLiveStrategy(
		ctx,
		&Env{
			L1Client:     client,
			Logger:       lgr,
			L1ScriptHost: host,
		},
		intent,
		st,
	)
	require.NoError(t, err)

	// Verify state was populated
	require.NotNil(t, st.SuperchainDeployment, "SuperchainDeployment should be populated")
	require.NotNil(t, st.SuperchainRoles, "SuperchainRoles should be populated")

	// Verify ProtocolVersions fields are zeroed for v2
	require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for v2")
	require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for v2")
	require.Equal(t, common.Address{}, st.SuperchainRoles.ProtocolVersionsOwner, "ProtocolVersionsOwner should be zero for v2")

	// Verify SuperchainConfig fields are populated
	require.Equal(t, superchain.SuperchainConfigAddr, st.SuperchainDeployment.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainProxyAdminImpl)
}

// Validates that providing both
// SuperchainConfigProxy and SuperchainRoles with opcmV2Enabled returns an error.
func TestInitLiveStrategy_OPCMV2WithSuperchainConfigProxyAndRoles_reverts(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	l1ChainID := uint64(11155111)
	superchain, err := standard.SuperchainFor(l1ChainID)
	require.NoError(t, err)

	// Set opcmV2Enabled flag via devFeatureBitmap
	opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
	intent := &state.Intent{
		ConfigType:            state.IntentTypeStandard,
		L1ChainID:             l1ChainID,
		L1ContractsLocator:    artifacts.EmbeddedLocator,
		L2ContractsLocator:    artifacts.EmbeddedLocator,
		SuperchainConfigProxy: &superchain.SuperchainConfigAddr,
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainGuardian: common.Address{0: 99},
		},
		GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": opcmV2Flag,
		},
	}

	st := &state.State{
		Version: 1,
	}

	err = InitLiveStrategy(
		ctx,
		&Env{
			L1Client: client,
			Logger:   lgr,
		},
		intent,
		st,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot set superchain roles when using predeployed OPCM or SuperchainConfig")
}

// Validates that providing both OPCMAddress and SuperchainConfigProxy works correctly
// The script will use the OPCM's semver to determine the version
func TestInitLiveStrategy_OPCMV1WithSuperchainConfigProxy(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	l1ChainID := uint64(11155111)
	superchain, err := standard.SuperchainFor(l1ChainID)
	require.NoError(t, err)

	opcmAddr, err := standard.OPCMImplAddressFor(l1ChainID, standard.CurrentTag)
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		afacts,
		rpcClient,
	)
	require.NoError(t, err)

	// Provide both OPCM address and SuperchainConfigProxy
	// The script will check the OPCM version and handle accordingly
	intent := &state.Intent{
		ConfigType:            state.IntentTypeStandard,
		L1ChainID:             l1ChainID,
		L1ContractsLocator:    artifacts.EmbeddedLocator,
		L2ContractsLocator:    artifacts.EmbeddedLocator,
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchain.SuperchainConfigAddr,
	}

	st := &state.State{
		Version: 1,
	}

	err = InitLiveStrategy(
		ctx,
		&Env{
			L1Client:     client,
			Logger:       lgr,
			L1ScriptHost: host,
		},
		intent,
		st,
	)
	// Should succeed - the script handles version detection
	require.NoError(t, err)

	// For OPCMv1, ProtocolVersions should be populated
	require.NotNil(t, st.SuperchainDeployment)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl)
}

// Validates that providing both
// OPCMAddress and SuperchainRoles with opcmV2Enabled=false returns an error.
func TestInitLiveStrategy_OPCMV1WithSuperchainRoles_reverts(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	l1ChainID := uint64(11155111)
	opcmAddr, err := standard.OPCMImplAddressFor(l1ChainID, standard.CurrentTag)
	require.NoError(t, err)

	// Don't set opcmV2Enabled flag (defaults to false)
	intent := &state.Intent{
		ConfigType:         state.IntentTypeStandard,
		L1ChainID:          l1ChainID,
		L1ContractsLocator: artifacts.EmbeddedLocator,
		L2ContractsLocator: artifacts.EmbeddedLocator,
		OPCMAddress:        &opcmAddr,
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainGuardian: common.Address{0: 99},
		},
	}

	st := &state.State{
		Version: 1,
	}

	err = InitLiveStrategy(
		ctx,
		&Env{
			L1Client: client,
			Logger:   lgr,
		},
		intent,
		st,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot set superchain roles when using predeployed OPCM or SuperchainConfig")
}

// Validates that the correct flow is chosen when
// hasPredeployedOPCM && !opcmV2Enabled, and that PopulateSuperchainState is called with correct parameters.
func TestInitLiveStrategy_FlowSelection_OPCMV1(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	l1ChainID := uint64(11155111)
	opcmAddr, err := standard.OPCMImplAddressFor(l1ChainID, standard.CurrentTag)
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		afacts,
		rpcClient,
	)
	require.NoError(t, err)

	// Don't set opcmV2Enabled flag (defaults to false)
	intent := &state.Intent{
		ConfigType:         state.IntentTypeStandard,
		L1ChainID:          l1ChainID,
		L1ContractsLocator: artifacts.EmbeddedLocator,
		L2ContractsLocator: artifacts.EmbeddedLocator,
		OPCMAddress:        &opcmAddr,
	}

	st := &state.State{
		Version: 1,
	}

	err = InitLiveStrategy(
		ctx,
		&Env{
			L1Client:     client,
			Logger:       lgr,
			L1ScriptHost: host,
		},
		intent,
		st,
	)
	require.NoError(t, err)

	// Verify OPCM v1 flow was used - ProtocolVersions should be populated
	require.NotNil(t, st.SuperchainDeployment)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy, "ProtocolVersionsProxy should be populated for v1")
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl, "ProtocolVersionsImpl should be populated for v1")
	require.NotEqual(t, common.Address{}, st.SuperchainRoles.ProtocolVersionsOwner, "ProtocolVersionsOwner should be populated for v1")

	// Verify ImplementationsDeployment was set
	require.NotNil(t, st.ImplementationsDeployment)
	require.Equal(t, opcmAddr, st.ImplementationsDeployment.OpcmImpl)
}

// Validates that the correct flow is chosen when
// hasSuperchainConfigProxy && opcmV2Enabled, and that PopulateSuperchainState is called with correct parameters.
func TestInitLiveStrategy_FlowSelection_OPCMV2(t *testing.T) {
	t.Parallel()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "SEPOLIA_RPC_URL must be set")

	lgr := testlog.Logger(t, slog.LevelInfo)
	retryProxy := devnet.NewRetryProxy(lgr, rpcURL)
	require.NoError(t, retryProxy.Start())
	t.Cleanup(func() {
		require.NoError(t, retryProxy.Stop())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(retryProxy.Endpoint())
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	l1ChainID := uint64(11155111)
	superchain, err := standard.SuperchainFor(l1ChainID)
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		afacts,
		rpcClient,
	)
	require.NoError(t, err)

	// Set opcmV2Enabled flag via devFeatureBitmap
	opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
	intent := &state.Intent{
		ConfigType:            state.IntentTypeStandard,
		L1ChainID:             l1ChainID,
		L1ContractsLocator:    artifacts.EmbeddedLocator,
		L2ContractsLocator:    artifacts.EmbeddedLocator,
		SuperchainConfigProxy: &superchain.SuperchainConfigAddr,
		GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": opcmV2Flag,
		},
	}

	st := &state.State{
		Version: 1,
	}

	err = InitLiveStrategy(
		ctx,
		&Env{
			L1Client:     client,
			Logger:       lgr,
			L1ScriptHost: host,
		},
		intent,
		st,
	)
	require.NoError(t, err)

	// Verify OPCM v2 flow was used - ProtocolVersions should be zeroed
	require.NotNil(t, st.SuperchainDeployment)
	require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for v2")
	require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for v2")
	require.Equal(t, common.Address{}, st.SuperchainRoles.ProtocolVersionsOwner, "ProtocolVersionsOwner should be zero for v2")

	// Verify SuperchainConfig is populated
	require.Equal(t, superchain.SuperchainConfigAddr, st.SuperchainDeployment.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)
}
