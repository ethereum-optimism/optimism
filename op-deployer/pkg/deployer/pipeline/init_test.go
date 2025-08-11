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
				SuperchainConfigImpl:     common.HexToAddress("0xCe28685EB204186b557133766eCA00334EB441E4"),
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
	dep, roles, err := PopulateSuperchainState(host, common.Address(*opcmAddr))
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
}
