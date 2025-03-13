package pipeline

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

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

	client, err := ethclient.Dial(retryProxy.Endpoint())
	require.NoError(t, err)

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

	t.Run("tagged L1 locator with standard intent types and standard roles", func(t *testing.T) {
		runTest := func(configType state.IntentType) {
			stdSuperchainRoles, err := state.GetStandardSuperchainRoles(l1ChainID)
			require.NoError(t, err)

			intent := &state.Intent{
				ConfigType:         configType,
				L1ChainID:          l1ChainID,
				L1ContractsLocator: artifacts.DefaultL1ContractsLocator,
				L2ContractsLocator: artifacts.DefaultL2ContractsLocator,
				SuperchainRoles:    stdSuperchainRoles,
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

			// Defining a file locator will always deploy a new superchain and OPCM
			superCfg, err := standard.SuperchainFor(l1ChainID)
			require.NoError(t, err)
			proxyAdmin, err := standard.SuperchainProxyAdminAddrFor(l1ChainID)
			require.NoError(t, err)
			opcmAddr, err := standard.ManagerImplementationAddrFor(l1ChainID, intent.L1ContractsLocator.Tag)
			require.NoError(t, err)

			expDeployment := &state.SuperchainDeployment{
				ProxyAdminAddress:            proxyAdmin,
				ProtocolVersionsProxyAddress: superCfg.ProtocolVersionsAddr,
				SuperchainConfigProxyAddress: superCfg.SuperchainConfigAddr,
			}

			// Tagged locator will reuse the existing superchain and OPCM
			require.NotNil(t, st.SuperchainDeployment)
			require.NotNil(t, st.ImplementationsDeployment)
			require.Equal(t, *expDeployment, *st.SuperchainDeployment)
			require.Equal(t, opcmAddr, st.ImplementationsDeployment.OpcmAddress)
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
				SuperchainRoles: &state.SuperchainRoles{
					Guardian: common.Address{0: 99},
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
			SuperchainRoles: &state.SuperchainRoles{
				Guardian: common.Address{0: 99},
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
