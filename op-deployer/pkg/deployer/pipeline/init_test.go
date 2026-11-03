package pipeline

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	opbindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestValidateInputsRejectsSP1VerifierWithPredeployedOPCM(t *testing.T) {
	_, _, dk := shared.DefaultPrivkey(t)
	l1ChainID := big.NewInt(900)
	loc := artifacts.MustNewLocatorFromURL("file:///test-artifacts")
	intent, st := shared.NewIntent(t, l1ChainID, dk, uint256.NewInt(1), loc, loc, standard.GasLimit)

	opcmAddress := common.Address{0x06}
	intent.OPCMAddress = &opcmAddress
	intent.SuperchainRoles = nil
	intent.GlobalDeployOverrides = map[string]any{
		"sp1Verifier": common.Address{0x05},
	}

	err := ValidateInputs(intent, st)
	require.ErrorContains(t, err, "sp1Verifier must not be specified when using a predeployed OPCM")
}

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
			proxyAdminContract, err := opbindings.NewProxyAdmin(proxyAdmin, client)
			require.NoError(t, err)
			superchainConfigImpl, err := proxyAdminContract.GetProxyImplementation(
				&bind.CallOpts{Context: ctx},
				superCfg.SuperchainConfigAddr,
			)
			require.NoError(t, err)

			expDeployment := &addresses.SuperchainContracts{
				SuperchainProxyAdminImpl: proxyAdmin,
				SuperchainConfigProxy:    superCfg.SuperchainConfigAddr,
				SuperchainConfigImpl:     superchainConfigImpl,
			}

			// Tagged locator will reuse the existing superchain and OPCM
			require.NotNil(t, st.SuperchainDeployment)
			require.NotNil(t, st.ImplementationsDeployment)
			require.NotNil(t, st.SuperchainRoles)
			require.Equal(t, *expDeployment, *st.SuperchainDeployment)
			require.Equal(t, opcmAddr, st.ImplementationsDeployment.OpcmV2Impl)
			require.Equal(t, stdSuperchainRoles.SuperchainProxyAdminOwner, st.SuperchainRoles.SuperchainProxyAdminOwner)
			require.Equal(t, stdSuperchainRoles.SuperchainGuardian, st.SuperchainRoles.SuperchainGuardian)
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

// TestPopulateSuperchainState reads the Sepolia superchain at a pinned block, so that upgrades on
// Sepolia don't change the expected values. Some of them, like the Superchain PAO, are not in the
// superchain registry and are therefore hardcoded.
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
	// The latest block on Sepolia as of 04/30/2025.
	client := pinnedCaller{client: ethclient.NewClient(rpcClient), block: big.NewInt(8227159)}

	superchain, err := standard.SuperchainFor(11155111)
	require.NoError(t, err)

	t.Run("Sepolia superchain", func(t *testing.T) {
		dep, roles, err := PopulateSuperchainState(context.Background(), client, superchain.SuperchainConfigAddr)
		require.NoError(t, err)
		require.Equal(t, addresses.SuperchainContracts{
			SuperchainProxyAdminImpl: common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"),
			SuperchainConfigProxy:    superchain.SuperchainConfigAddr,
			SuperchainConfigImpl:     common.HexToAddress("0x4da82a327773965b8d4D85Fa3dB8249b387458E7"),
		}, *dep)
		require.Equal(t, addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2"),
			SuperchainGuardian:        common.HexToAddress("0x7a50f00e8D05b95F98fE38d8BeE366a7324dCf7E"),
		}, *roles)
	})

	t.Run("proxy without code", func(t *testing.T) {
		for _, proxy := range []common.Address{{}, common.HexToAddress("0x1234567890123456789012345678901234567890")} {
			dep, roles, err := PopulateSuperchainState(context.Background(), client, proxy)
			require.ErrorContains(t, err, "has no code", "proxy %s", proxy)
			require.Nil(t, dep)
			require.Nil(t, roles)
		}
	})
}

// pinnedCaller reads the state of one block.
type pinnedCaller struct {
	client *ethclient.Client
	block  *big.Int
}

func (c pinnedCaller) CodeAt(ctx context.Context, contract common.Address, _ *big.Int) ([]byte, error) {
	return c.client.CodeAt(ctx, contract, c.block)
}

func (c pinnedCaller) CallContract(ctx context.Context, call ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	return c.client.CallContract(ctx, call, c.block)
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

	intent := &state.Intent{
		ConfigType:            state.IntentTypeStandard,
		L1ChainID:             l1ChainID,
		L1ContractsLocator:    artifacts.EmbeddedLocator,
		L2ContractsLocator:    artifacts.EmbeddedLocator,
		SuperchainConfigProxy: &superchain.SuperchainConfigAddr,
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
	require.NoError(t, err)

	// Verify state was populated
	require.NotNil(t, st.SuperchainDeployment, "SuperchainDeployment should be populated")
	require.NotNil(t, st.SuperchainRoles, "SuperchainRoles should be populated")

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

	intent := &state.Intent{
		ConfigType:            state.IntentTypeStandard,
		L1ChainID:             l1ChainID,
		L1ContractsLocator:    artifacts.EmbeddedLocator,
		L2ContractsLocator:    artifacts.EmbeddedLocator,
		SuperchainConfigProxy: &superchain.SuperchainConfigAddr,
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

// Validates that providing both OPCMAddress and SuperchainConfigProxy works correctly.
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

	// Provide both OPCM address and SuperchainConfigProxy
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
			L1Client: client,
			Logger:   lgr,
		},
		intent,
		st,
	)
	require.NoError(t, err)

	require.NotNil(t, st.SuperchainDeployment)
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
// hasPredeployedOPCM && !opcmV2Enabled, and that the superchain deployment is read.
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
			L1Client: client,
			Logger:   lgr,
		},
		intent,
		st,
	)
	require.NoError(t, err)

	require.NotNil(t, st.SuperchainDeployment)

	// Verify ImplementationsDeployment was set
	require.NotNil(t, st.ImplementationsDeployment)
	require.Equal(t, opcmAddr, st.ImplementationsDeployment.OpcmV2Impl)
}

// TestInitLiveStrategy_CompletesRecordedOPCM checks that init reads the implementations of a
// predeployed OPCM when the state records only the OPCM, as an earlier op-deployer left it.
func TestInitLiveStrategy_CompletesRecordedOPCM(t *testing.T) {
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

	intent := &state.Intent{
		ConfigType:         state.IntentTypeStandard,
		L1ChainID:          l1ChainID,
		L1ContractsLocator: artifacts.EmbeddedLocator,
		L2ContractsLocator: artifacts.EmbeddedLocator,
		OPCMAddress:        &opcmAddr,
	}
	st := &state.State{
		Version:                   1,
		ImplementationsDeployment: &addresses.ImplementationsContracts{OpcmV2Impl: opcmAddr},
	}
	require.NoError(t, InitLiveStrategy(ctx, &Env{L1Client: client, Logger: lgr}, intent, st))

	impls, err := opcm.ReadImplementations(ctx, client, opcmAddr)
	require.NoError(t, err)
	require.Equal(t, impls, st.ImplementationsDeployment)
}

// Validates that the correct flow is chosen when
// hasSuperchainConfigProxy && opcmV2Enabled, and that the superchain deployment is read.
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

	intent := &state.Intent{
		ConfigType:            state.IntentTypeStandard,
		L1ChainID:             l1ChainID,
		L1ContractsLocator:    artifacts.EmbeddedLocator,
		L2ContractsLocator:    artifacts.EmbeddedLocator,
		SuperchainConfigProxy: &superchain.SuperchainConfigAddr,
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
	require.NoError(t, err)

	// Verify OPCM v2 flow was used
	require.NotNil(t, st.SuperchainDeployment)

	// Verify SuperchainConfig is populated
	require.Equal(t, superchain.SuperchainConfigAddr, st.SuperchainDeployment.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)
}
