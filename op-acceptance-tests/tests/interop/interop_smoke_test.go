package interop

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	sdktypes "github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func smokeTestScenario(chainIdx uint64, walletGetter validators.WalletGetter) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()

		logger := testlog.Logger(t, log.LevelInfo)
		logger = logger.With("test", "TestMinimal", "devnet", sys.Identifier())

		chain := sys.L2s()[chainIdx]
		logger = logger.With("chain", chain.ID())
		logger.Info("starting test")

		funds := sdktypes.NewBalance(big.NewInt(0.5 * constants.ETH))
		user := walletGetter(ctx)

		scw0Addr := constants.SuperchainWETH
		scw0, err := chain.ContractsRegistry().SuperchainWETH(scw0Addr)
		require.NoError(t, err)
		logger.Info("using SuperchainWETH", "contract", scw0Addr)

		initialBalance, err := scw0.BalanceOf(user.Address()).Call(ctx)
		require.NoError(t, err)
		logger = logger.With("user", user.Address())
		logger.Info("initial balance retrieved", "balance", initialBalance)

		logger.Info("sending ETH to contract", "amount", funds)
		require.NoError(t, user.SendETH(scw0Addr, funds).Send(ctx).Wait())

		balance, err := scw0.BalanceOf(user.Address()).Call(ctx)
		require.NoError(t, err)
		logger.Info("final balance retrieved", "balance", balance)

		require.Equal(t, initialBalance.Add(funds), balance)
	}
}

func TestSystemWrapETH(t *testing.T) {
	chainIdx := uint64(0) // We'll use the first L2 chain for this test

	walletGetter, fundsValidator := validators.AcquireL2WalletWithFunds(chainIdx, sdktypes.NewBalance(big.NewInt(1.0*constants.ETH)))
	_, interopValidator := validators.AcquireL2WithFork(chainIdx, rollup.Interop)

	systest.SystemTest(t,
		smokeTestScenario(chainIdx, walletGetter),
		fundsValidator,
		interopValidator,
	)
}

func TestInteropSystemNoop(t *testing.T) {
	systest.InteropSystemTest(t, func(t systest.T, sys system.InteropSystem) {
		testlog.Logger(t, log.LevelInfo).Info("noop")
	})
}

func TestInteropSystemSupervisor(t *testing.T) {
	systest.InteropSystemTest(t, func(t systest.T, sys system.InteropSystem) {
		ctx := t.Context()
		supervisor, err := sys.Supervisor(ctx)
		require.NoError(t, err)
		block, err := supervisor.FinalizedL1(ctx)
		require.NoError(t, err)
		require.NotNil(t, block)
		testlog.Logger(t, log.LevelInfo).Info("finalized l1 block", "block", block)
	})
}

func TestSmokeTestFailure(t *testing.T) {
	// Create mock failing system
	mockAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	mockWallet := &mockFailingWallet{
		addr: mockAddr,
		bal:  sdktypes.NewBalance(big.NewInt(1000000)),
	}
	mockL1Chain := newMockFailingL1Chain(
		sdktypes.ChainID(big.NewInt(1234)),
		system.WalletMap{
			"user1": mockWallet,
		},
	)
	mockL2Chain := newMockFailingL2Chain(
		sdktypes.ChainID(big.NewInt(1234)),
		system.WalletMap{"user1": mockWallet},
	)
	mockSys := &mockFailingSystem{l1Chain: mockL1Chain, l2Chain: mockL2Chain}

	// Run the smoke test logic and capture failures
	getter := func(ctx context.Context) system.Wallet {
		return mockWallet
	}
	rt := NewRecordingT(context.TODO())
	rt.TestScenario(
		smokeTestScenario(0, getter),
		mockSys,
	)

	// Verify that the test failed due to SendETH error
	require.True(t, rt.Failed(), "test should have failed")
	require.Contains(t, rt.Logs(), "transaction failure", "unexpected failure message")
}

func lowLevelSystemScenario(sysGetter validators.LowLevelSystemGetter) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		logger := testlog.Logger(t, log.LevelInfo).With("test", "TestLowLevelSystem", "devnet", sys.Identifier())
		_ = sysGetter(t.Context())
		logger.Info("low level system acquired")
	}
}

func TestLowLevelSystem(t *testing.T) {
	lowLevelSys, validator := validators.AcquireLowLevelSystem()
	systest.SystemTest(t,
		lowLevelSystemScenario(lowLevelSys),
		validator,
	)
}
