package operatorfee

import (
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestFees verifies that L1/L2 fees are handled properly in different fork configurations
func TestOperatorFee(t *testing.T) {
	logger := testlog.Logger(t, slog.LevelDebug)
	// Define which L2 chain we'll test
	chainIdx := uint64(0)

	logger.Info("Starting operator fee test", "chain", chainIdx)

	// Get validators and getters for accessing the system and wallets
	l1WalletGetter, l1WalletValidator := validators.AcquireL1WalletWithFunds(types.NewBalance(big.NewInt(params.Ether)))
	l2WalletGetter, l2WalletValidator := validators.AcquireL2WalletWithFunds(chainIdx, types.NewBalance(big.NewInt(params.Ether)))

	logger.Info("Acquired test wallets with funds")

	// Run isthmus test
	_, forkValidator := validators.AcquireL2WithFork(chainIdx, rollup.Isthmus)
	nodesValidator := validators.HasSufficientL2Nodes(chainIdx, 2)
	logger.Info("Running system test", "fork", "Isthmus", "nodes", 2)
	systest.SystemTest(t,
		func(t systest.T, sys system.System) {
			logger.Info("Starting operator fee test scenario", "chain", chainIdx)
			// Get the low-level system and wallet
			l1Wallet := l1WalletGetter(t.Context())
			l2Wallet := l2WalletGetter(t.Context())
			logger.Info("Acquired wallets",
				"l1_wallet", l1Wallet.Address().Hex(),
				"l2_wallet", l2Wallet.Address().Hex())

			// get l2WalletBalance
			l2GethSeqClient, err := sys.L2s()[chainIdx].Nodes()[0].GethClient()
			require.NoError(t, err)
			l2WalletBalance, err := l2GethSeqClient.BalanceAt(t.Context(), l2Wallet.Address(), nil)
			require.NoError(t, err)
			logger.Info("L2 wallet balance", "balance", l2WalletBalance)

			// Define test cases with different operator fee parameters
			numRandomValuesForEachDimm := 1
			testCases := GenerateAllTestParamsCases(numRandomValuesForEachDimm)

			// For each test case, verify the operator fee parameters
			for _, tc := range testCases {
				t.Run(tc.ID, func(t systest.T) {
					operatorFeeTestProcedure(t, sys, l1Wallet, l2Wallet, chainIdx, tc, logger)
				})
			}
		},
		l2WalletValidator,
		l1WalletValidator,
		forkValidator,
		nodesValidator,
	)
}

func operatorFeeTestProcedure(t systest.T, sys system.System, l1FundingWallet system.Wallet, l2FundingWallet system.Wallet, chainIdx uint64, tc TestParams, logger log.Logger) {
	ctx := t.Context()
	logger.Info("Starting operator fee test",
		"test_case", tc.ID,
		"operator_fee_constant", tc.OperatorFeeConstant,
		"operator_fee_scalar", tc.OperatorFeeScalar,
		"l1_fee_constant", tc.L1BlobBaseFeeScalar,
		"l1_fee_scalar", tc.L1BaseFeeScalar,
	)

	// ==========
	// Read-only Test Setup + Invariant Checks
	// ==========

	// Setup clients
	logger.Info("Setting up clients for L1 and L2 chains")
	l1GethClient, err := sys.L1().Nodes()[0].GethClient()
	require.NoError(t, err)
	l2Chain := sys.L2s()[chainIdx]
	l2GethSeqClient, err := l2Chain.Nodes()[0].GethClient()
	require.NoError(t, err)

	// Setup chain fork detection
	secondCheck, err := systest.CheckForChainFork(t.Context(), l2Chain, logger)
	require.NoError(t, err, "error checking for chain fork")
	defer func() {
		require.NoError(t, secondCheck(), "error checking for chain fork")
	}()

	l2StartHeader, err := l2GethSeqClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	// Verify coinbase address is the same as the sequencer fee vault address
	require.Equal(t, l2StartHeader.Coinbase, predeploys.SequencerFeeVaultAddr, "coinbase address should always be the same as the sequencer fee vault address")

	// setup rollup owner wallet
	logger.Info("Setting up rollup owner wallet")
	l1RollupOwnerWallet, ok := sys.L2s()[chainIdx].L1Wallets()["systemConfigOwner"]

	require.True(t, ok, "rollup owner wallet not found")
	require.NotNil(t, l1RollupOwnerWallet, "rollup owner wallet not found")

	l1ChainID, err := l1GethClient.ChainID(ctx)
	require.NoError(t, err)
	logger.Debug("L1 chain ID", "chainID", l1ChainID)

	// Get the genesis config
	logger.Info("Getting L2 chain config")
	l2ChainConfig, err := l2Chain.Config()
	require.NoError(t, err)

	l2ChainID := l2ChainConfig.ChainID

	// Create fee checker
	logger.Info("Creating fee checker utility")
	feeChecker := NewFeeChecker(t, l2GethSeqClient, l2ChainConfig, logger)

	// Setup GasPriceOracle contract binding
	logger.Info("Connecting to GasPriceOracle contract")
	gpoContract, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, l2GethSeqClient)
	require.NoError(t, err)

	// Setup L2 L1Block contract binding
	l2L1BlockContract, err := bindings.NewL1Block(predeploys.L1BlockAddr, l2GethSeqClient)
	require.NoError(t, err)

	// Initialize systemconfig contract
	logger.Info("Getting SystemConfig contract")
	systemConfigProxyAddr, ok := l2Chain.L1Addresses()["systemConfigProxy"]
	require.True(t, ok, "system config proxy address not found")
	systemConfig, err := bindings.NewSystemConfig(systemConfigProxyAddr, l1GethClient)
	require.NoError(t, err)

	// Verify system config proxy owner is the rollup owner
	owner, err := systemConfig.Owner(&bind.CallOpts{BlockNumber: nil})
	require.NoError(t, err)
	require.Equal(t, owner, l1RollupOwnerWallet.Address(), "system config proxy owner should be the rollup owner")

	// Verify GPO isthmus view matches chain isthmus view
	gpoIsthmus, err := gpoContract.IsIsthmus(&bind.CallOpts{BlockNumber: l2StartHeader.Number})
	require.NoError(t, err)
	require.True(t, gpoIsthmus, "GPO and chain must have same isthmus view")
	logger.Info("Verified GPO contract has correct Isthmus view")

	// Create balance reader
	logger.Info("Creating balance reader")
	balanceReader := NewBalanceReader(t, l2GethSeqClient, logger)

	// Wait for first block after genesis. The genesis block has zero L1Block
	// values and will throw off the GPO checks
	logger.Info("Waiting for L2 chain to produce block 1")
	_, err = l2GethSeqClient.HeaderByNumber(ctx, big.NewInt(1))
	require.NoError(t, err)

	// Create test wallets
	logger.Info("Creating test wallet 1")
	l2TestWallet1, err := NewTestWallet(ctx, l2Chain)
	require.NoError(t, err)
	logger.Info("Test wallet 1", "address", l2TestWallet1.Address().Hex())

	logger.Info("Creating test wallet 2")
	l2TestWallet2, err := NewTestWallet(ctx, l2Chain)
	require.NoError(t, err)
	logger.Info("Test wallet 2", "address", l2TestWallet2.Address().Hex())

	fundAmount := big.NewInt(1e18)

	// ==========
	// Begin Test
	// ==========

	// Fund l1RollupOwnerWallet wallet from faucet
	logger.Info("Funding rollup owner wallet with 10 ETH")
	_, _, err = SendValueTx(ctx, l1ChainID, l1GethClient, l1FundingWallet, l1RollupOwnerWallet.Address(), new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(10)), true)
	require.NoError(t, err, "Error funding owner wallet")
	defer func() {
		ReturnRemainingFunds(t, ctx, l1ChainID, l1GethClient, l1RollupOwnerWallet, l1FundingWallet, logger)
	}()

	// Fund test wallet from faucet
	logger.Info("Funding test wallet with ETH", "amount", fundAmount)
	_, _, err = SendValueTx(ctx, l2ChainID, l2GethSeqClient, l2FundingWallet, l2TestWallet1.Address(), fundAmount, true)
	require.NoError(t, err, "Error funding test wallet")
	defer func() {
		ReturnRemainingFunds(t, ctx, l2ChainID, l2GethSeqClient, l2TestWallet1, l2FundingWallet, logger)
	}()

	// check that the balance of l2TestWallet1 is now the fund amount
	balance, err := l2GethSeqClient.BalanceAt(ctx, l2TestWallet1.Address(), nil)
	require.NoError(t, err)
	require.Equal(t, fundAmount, balance, "balance of l2TestWallet1 should be the fund amount")

	// Update operator fee parameters
	logger.Info("Updating operator fee parameters",
		"constant", tc.OperatorFeeConstant,
		"scalar", tc.OperatorFeeScalar)
	_, receipt := UpdateOperatorFeeParams(t, l1ChainID, l1GethClient, systemConfig, systemConfigProxyAddr, l1RollupOwnerWallet, tc.OperatorFeeConstant, tc.OperatorFeeScalar, logger)
	logger.Info("Operator fee parameters updated", "block", receipt.BlockNumber)

	// Update L1 fee parameters
	logger.Info("Updating L1 fee parameters",
		"l1BaseFeeScalar", tc.L1BaseFeeScalar,
		"l1BlobBaseFeeScalar", tc.L1BlobBaseFeeScalar)
	_, _ = UpdateL1FeeParams(t, l1ChainID, l1GethClient, systemConfig, systemConfigProxyAddr, l1RollupOwnerWallet, tc.L1BaseFeeScalar, tc.L1BlobBaseFeeScalar, logger)
	logger.Info("Operator fee parameters updated", "block", receipt.BlockNumber)

	// sleep to allow for the L2 nodes to sync to L1 origin where operator fee was set
	delay := 30 * time.Second
	logger.Info("Waiting for L2 nodes to sync with L1 origin where operator fee was set", "delay", delay)
	time.Sleep(delay)

	// Verify L1Block contract values have been updated to match test case values
	baseFeeScalar, err := l2L1BlockContract.BaseFeeScalar(&bind.CallOpts{BlockNumber: nil})
	require.NoError(t, err)
	logger.Info("L1Block base fee scalar", "scalar", baseFeeScalar)
	require.Equal(t, tc.L1BaseFeeScalar, baseFeeScalar, "L1Block base fee scalar does not match test case value")

	blobBaseFeeScalar, err := l2L1BlockContract.BlobBaseFeeScalar(&bind.CallOpts{BlockNumber: nil})
	require.NoError(t, err)
	logger.Info("L1Block blob base fee scalar", "scalar", blobBaseFeeScalar)
	require.Equal(t, tc.L1BlobBaseFeeScalar, blobBaseFeeScalar, "L1Block blob base fee scalar does not match test case value")

	operatorFeeConstant, err := l2L1BlockContract.OperatorFeeConstant(&bind.CallOpts{BlockNumber: nil})
	require.NoError(t, err)
	logger.Info("L1Block operator fee constant", "constant", operatorFeeConstant)
	require.Equal(t, tc.OperatorFeeConstant, operatorFeeConstant, "L1Block operator fee constant does not match test case value")

	operatorFeeScalar, err := l2L1BlockContract.OperatorFeeScalar(&bind.CallOpts{BlockNumber: nil})
	require.NoError(t, err)
	logger.Info("L1Block operator fee scalar", "scalar", operatorFeeScalar)
	require.Equal(t, tc.OperatorFeeScalar, operatorFeeScalar, "L1Block operator fee scalar does not match test case value")

	l2PreTestHeader, err := l2GethSeqClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	// Get initial balances
	logger.Info("Sampling initial balances", "block", l2PreTestHeader.Number.Uint64())
	startBalances := balanceReader.SampleBalances(ctx, l2PreTestHeader.Number, l2TestWallet1.Address())
	logger.Debug("Initial balances", "balances", startBalances)

	// Send the test transaction
	logger.Info("Current base fee", "fee", l2PreTestHeader.BaseFee)
	receipt, tx, err := SendValueTx(ctx, l2ChainID, l2GethSeqClient, l2TestWallet1, l2TestWallet2.Address(), big.NewInt(1000), true)

	defer func() {
		ReturnRemainingFunds(t, ctx, l2ChainID, l2GethSeqClient, l2TestWallet1, l2FundingWallet, logger)
		ReturnRemainingFunds(t, ctx, l2ChainID, l2GethSeqClient, l2TestWallet2, l2FundingWallet, logger)
	}()

	require.NoError(t, err, "failed to send test transaction where it should succeed")
	logger.Info("Transaction confirmed",
		"block", receipt.BlockNumber.Uint64(),
		"hash", tx.Hash().Hex())

	// Get final balances after transaction
	logger.Info("Sampling final balances", "block", receipt.BlockNumber.Uint64())
	endBalances := balanceReader.SampleBalances(ctx, receipt.BlockNumber, l2TestWallet1.Address())
	logger.Debug("Final balances", "balances", endBalances)

	// Calculate L1 fee for GPO verification
	l2EndHeader, err := l2GethSeqClient.HeaderByNumber(ctx, receipt.BlockNumber)
	require.NoError(t, err)
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err)
	l1Fee := feeChecker.L1Cost(tx.RollupCostData(), l2EndHeader.Time)
	logger.Debug("Calculated L1 fee", "fee", l1Fee)

	// Verify gas price oracle L1 fee calculation
	adjustedGPOFee, err := gpoContract.GetL1Fee(&bind.CallOpts{BlockNumber: receipt.BlockNumber}, txBytes)
	require.NoError(t, err)
	logger.Debug("GPO contract L1 fee", "fee", adjustedGPOFee)
	// Verify that GPO contract L1 fee calculation matches local L1 fee calculation
	require.Equal(t, l1Fee, adjustedGPOFee, "GPO reports L1 fee mismatch")
	// Verify execution L1 fee calculation matches GPO and local L1 fee calculation
	require.Equal(t, l1Fee, receipt.L1Fee, "l1 fee in receipt is correct")

	// Calculate expected fee changes from raw inputs
	logger.Info("Calculating expected balance changes based on transaction data")
	expectedChanges := feeChecker.CalculateExpectedBalanceChanges(
		receipt.GasUsed,
		l2EndHeader,
		tx,
	)
	logger.Debug("Expected balance changes", "changes", expectedChanges)

	// Calculate expected end balances using the new method
	expectedEndBalances := startBalances.Add(expectedChanges)
	expectedEndBalances.BlockNumber = l2EndHeader.Number
	logger.Debug("Expected final balances", "balances", expectedEndBalances)

	// Assert that actual end balances match what we calculated
	logger.Info("Verifying actual balances match expected balances")
	AssertSnapshotsEqual(t, expectedEndBalances, endBalances)
}
