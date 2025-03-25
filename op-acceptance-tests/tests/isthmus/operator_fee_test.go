package isthmus

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/assert"
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
		operatorFeeTestScenario(l1WalletGetter, l2WalletGetter, chainIdx, logger),
		l2WalletValidator,
		l1WalletValidator,
		forkValidator,
		nodesValidator,
	)
}

// stateGetterAdapter adapts the ethclient to implement the StateGetter interface
type stateGetterAdapter struct {
	t      systest.T
	client *ethclient.Client
	ctx    context.Context
}

// GetState implements the StateGetter interface
func (sga *stateGetterAdapter) GetState(addr common.Address, key common.Hash) common.Hash {
	var result common.Hash
	val, err := sga.client.StorageAt(sga.ctx, addr, key, nil)
	require.NoError(sga.t, err)
	copy(result[:], val)
	return result
}

// waitForTransaction polls for a transaction receipt until it is available or the context is canceled.
// It's a simpler version of the functionality in SimpleTxManager.
func waitForTransaction(ctx context.Context, client *ethclient.Client, hash common.Hash) (*gethTypes.Receipt, error) {
	ticker := time.NewTicker(500 * time.Millisecond) // Poll every 500ms
	defer ticker.Stop()

	// Record starting block number
	startBlockNum, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get starting block number: %w", err)
	}

	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if receipt != nil && err == nil {
			return receipt, nil
		} else if err != nil && !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
		}

		select {
		case <-ctx.Done():
			// Get current block number to calculate progress
			// Create a new context for this query since the original is canceled
			queryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			currentBlockNum, blockErr := client.BlockNumber(queryCtx)
			cancel() // Always cancel the context to avoid leaks

			if blockErr != nil {
				// If we can't get the current block number, just return the original error
				return nil, fmt.Errorf("context error: %w (could not determine block progress: %w)", ctx.Err(), blockErr)
			}

			blockProgress := int64(currentBlockNum) - int64(startBlockNum)
			return nil, fmt.Errorf("transaction %s not found after %d blocks: %w", hash.Hex(), blockProgress, ctx.Err())
		case <-ticker.C:
			// Continue polling
		}
	}
}

// BalanceSnapshot stores the balances of various fee recipients at a specific block
type BalanceSnapshot struct {
	BlockNumber         *big.Int
	BaseFeeVaultBalance *big.Int
	L1FeeVaultBalance   *big.Int
	SequencerFeeVault   *big.Int
	OperatorFeeVault    *big.Int
	FromBalance         *big.Int
}

// String returns a formatted string representation of the balance snapshot
func (bs *BalanceSnapshot) String() string {
	if bs == nil {
		return "nil"
	}

	return fmt.Sprintf(
		"BalanceSnapshot{Block: %v, BaseFeeVault: %v, L1FeeVault: %v, SequencerFeeVault: %v, "+
			"OperatorFeeVault: %v, WalletBalance: %v}",
		bs.BlockNumber,
		bs.BaseFeeVaultBalance,
		bs.L1FeeVaultBalance,
		bs.SequencerFeeVault,
		bs.OperatorFeeVault,
		bs.FromBalance,
	)
}

// Add adds this snapshot's balances to another snapshot and returns a new snapshot
// This is typically used to apply changes to a starting balance snapshot
func (bs *BalanceSnapshot) Add(start *BalanceSnapshot) *BalanceSnapshot {
	if bs == nil || start == nil {
		return nil
	}

	return &BalanceSnapshot{
		BlockNumber:         bs.BlockNumber, // Use the target block number from changes
		BaseFeeVaultBalance: new(big.Int).Add(start.BaseFeeVaultBalance, bs.BaseFeeVaultBalance),
		L1FeeVaultBalance:   new(big.Int).Add(start.L1FeeVaultBalance, bs.L1FeeVaultBalance),
		SequencerFeeVault:   new(big.Int).Add(start.SequencerFeeVault, bs.SequencerFeeVault),
		OperatorFeeVault:    new(big.Int).Add(start.OperatorFeeVault, bs.OperatorFeeVault),
		FromBalance:         new(big.Int).Add(start.FromBalance, bs.FromBalance),
	}
}

// Sub returns a new BalanceSnapshot containing the differences between this snapshot and another
// This snapshot is considered the "end" and the parameter is the "start"
// Positive values indicate increases, negative values indicate decreases
func (bs *BalanceSnapshot) Sub(start *BalanceSnapshot) *BalanceSnapshot {
	if bs == nil || start == nil {
		return nil
	}

	return &BalanceSnapshot{
		BlockNumber:         bs.BlockNumber,
		BaseFeeVaultBalance: new(big.Int).Sub(bs.BaseFeeVaultBalance, start.BaseFeeVaultBalance),
		L1FeeVaultBalance:   new(big.Int).Sub(bs.L1FeeVaultBalance, start.L1FeeVaultBalance),
		SequencerFeeVault:   new(big.Int).Sub(bs.SequencerFeeVault, start.SequencerFeeVault),
		OperatorFeeVault:    new(big.Int).Sub(bs.OperatorFeeVault, start.OperatorFeeVault),
		FromBalance:         new(big.Int).Sub(bs.FromBalance, start.FromBalance),
	}
}

// AssertSnapshotsEqual compares two balance snapshots and reports differences
func AssertSnapshotsEqual(t systest.T, expected, actual *BalanceSnapshot) {
	require.NotNil(t, expected, "Expected snapshot should not be nil")
	require.NotNil(t, actual, "Actual snapshot should not be nil")

	// Check base fee vault balance
	assert.True(t, expected.BaseFeeVaultBalance.Cmp(actual.BaseFeeVaultBalance) == 0,
		"BaseFeeVaultBalance mismatch: expected %v, got %v (diff: %v)", expected.BaseFeeVaultBalance, actual.BaseFeeVaultBalance, new(big.Int).Sub(actual.BaseFeeVaultBalance, expected.BaseFeeVaultBalance))

	// Check L1 fee vault balance
	assert.True(t, expected.L1FeeVaultBalance.Cmp(actual.L1FeeVaultBalance) == 0,
		"L1FeeVaultBalance mismatch: expected %v, got %v (diff: %v)", expected.L1FeeVaultBalance, actual.L1FeeVaultBalance, new(big.Int).Sub(actual.L1FeeVaultBalance, expected.L1FeeVaultBalance))

	// Check sequencer fee vault balance
	assert.True(t, expected.SequencerFeeVault.Cmp(actual.SequencerFeeVault) == 0,
		"SequencerFeeVault mismatch: expected %v, got %v (diff: %v)", expected.SequencerFeeVault, actual.SequencerFeeVault, new(big.Int).Sub(actual.SequencerFeeVault, expected.SequencerFeeVault))

	// Check operator fee vault balance
	assert.True(t, expected.OperatorFeeVault.Cmp(actual.OperatorFeeVault) == 0,
		"OperatorFeeVault mismatch: expected %v, got %v (diff: %v)", expected.OperatorFeeVault, actual.OperatorFeeVault, new(big.Int).Sub(actual.OperatorFeeVault, expected.OperatorFeeVault))

	// Check wallet balance
	assert.True(t, expected.FromBalance.Cmp(actual.FromBalance) == 0,
		"WalletBalance mismatch: expected %v, got %v (diff: %v)", expected.FromBalance, actual.FromBalance, new(big.Int).Sub(actual.FromBalance, expected.FromBalance))
}

// FeeChecker provides methods to calculate various types of fees
type FeeChecker struct {
	config        *params.ChainConfig
	l1CostFn      gethTypes.L1CostFunc
	operatorFeeFn gethTypes.OperatorCostFunc
	logger        log.Logger
}

// NewFeeChecker creates a new FeeChecker instance
func NewFeeChecker(t systest.T, client *ethclient.Client, chainConfig *params.ChainConfig, logger log.Logger) *FeeChecker {
	logger.Debug("Creating fee checker", "chainID", chainConfig.ChainID)
	// Create state getter adapter for L1 cost function
	sga := &stateGetterAdapter{
		t:      t,
		client: client,
		ctx:    t.Context(),
	}

	// Create L1 cost function
	l1CostFn := gethTypes.NewL1CostFunc(chainConfig, sga)

	// Create operator fee function
	operatorFeeFn := gethTypes.NewOperatorCostFunc(chainConfig, sga)

	return &FeeChecker{
		config:        chainConfig,
		l1CostFn:      l1CostFn,
		operatorFeeFn: operatorFeeFn,
		logger:        logger,
	}
}

// L1Cost calculates the L1 fee for a transaction
func (fc *FeeChecker) L1Cost(rcd gethTypes.RollupCostData, blockTime uint64) *big.Int {
	return fc.l1CostFn(rcd, blockTime)
}

// CalculateExpectedBalanceChanges creates a BalanceSnapshot containing expected fee movements
// Calculates all fees internally from raw inputs
func (fc *FeeChecker) CalculateExpectedBalanceChanges(
	receipt *gethTypes.Receipt,
	header *gethTypes.Header,
	tx *gethTypes.Transaction,
) *BalanceSnapshot {
	// Convert the gas used (uint64) to a big.Int.
	gasUsed := new(big.Int).SetUint64(receipt.GasUsed)

	// 1. Base Fee Burned: header.BaseFee * gasUsed
	baseFee := new(big.Int).Mul(header.BaseFee, gasUsed)

	// 2. Calculate the effective tip.
	// Effective tip is the minimum of:
	//   a) tx.GasTipCap() and
	//   b) tx.GasFeeCap() - header.BaseFee
	tipCap := tx.GasTipCap() // maximum priority fee per gas offered by the user
	feeCap := tx.GasFeeCap() // maximum fee per gas the user is willing to pay

	// Compute feeCap minus the base fee.
	diff := new(big.Int).Sub(feeCap, header.BaseFee)

	// effectiveTip = min(tipCap, diff)
	effectiveTip := new(big.Int)
	if tipCap.Cmp(diff) < 0 {
		effectiveTip.Set(tipCap)
	} else {
		effectiveTip.Set(diff)
	}

	// 3. Coinbase Fee Credit: effectiveTip * gasUsed.
	l2Fee := new(big.Int).Mul(effectiveTip, gasUsed)

	// Calculate L1 fee
	l1Fee := fc.L1Cost(tx.RollupCostData(), header.Time)

	// Calculate operator fee
	fc.logger.Debug("Calculating operator fee", "gasUsed", receipt.GasUsed, "blockTime", header.Time)
	operatorFee := fc.operatorFeeFn(receipt.GasUsed, header.Time).ToBig()

	txFeesAndValue := new(big.Int).Set(baseFee)
	txFeesAndValue.Add(txFeesAndValue, l2Fee)
	txFeesAndValue.Add(txFeesAndValue, l1Fee)
	txFeesAndValue.Add(txFeesAndValue, operatorFee)
	txFeesAndValue.Add(txFeesAndValue, tx.Value())

	// Create a changes snapshot with expected fee movements
	changes := &BalanceSnapshot{
		BaseFeeVaultBalance: baseFee,
		L1FeeVaultBalance:   l1Fee,
		SequencerFeeVault:   l2Fee,
		OperatorFeeVault:    operatorFee, // Operator fee is withdrawn
		FromBalance:         new(big.Int).Neg(txFeesAndValue),
	}

	return changes
}

func performOperatorFeeTest(t systest.T, sys system.System, l1FundingWallet system.Wallet, l2FundingWallet system.Wallet, chainIdx uint64, operatorFeeConstant uint64, operatorFeeScalar uint32, logger log.Logger) {
	ctx := t.Context()
	logger.Info("Starting operator fee test",
		"constant", operatorFeeConstant,
		"scalar", operatorFeeScalar)

	// Setup clients
	logger.Info("Setting up clients for L1 and L2 chains")
	l1GethClient, err := sys.L1().Nodes()[0].GethClient()
	require.NoError(t, err)
	l2Chain := sys.L2s()[chainIdx]
	l2GethSeqClient, err := l2Chain.Nodes()[0].GethClient()
	require.NoError(t, err)
	l2RethFullClient, err := l2Chain.Nodes()[1].GethClient()
	require.NoError(t, err)
	l2MultiClient := systest.NewMultiClient([]*ethclient.Client{l2GethSeqClient, l2RethFullClient})

	// Get the genesis config
	l1ChainID, err := l1GethClient.ChainID(ctx)
	require.NoError(t, err)
	logger.Debug("L1 chain ID", "chainID", l1ChainID)

	// Wait for first block after genesis
	// The genesis block has zero L1Block values and will throw off the GPO checks
	logger.Info("Waiting for L2 chain to produce block 1")
	_, err = l2GethSeqClient.HeaderByNumber(ctx, big.NewInt(1))
	require.NoError(t, err)

	// setup rollup owner wallet
	logger.Info("Setting up rollup owner wallet")
	l1RollupOwnerWallet, ok := sys.L2s()[chainIdx].L1Wallets()["systemConfigOwner"]

	require.True(t, ok, "rollup owner wallet not found")
	require.NotNil(t, l1RollupOwnerWallet, "rollup owner wallet not found")

	logger.Info("Funding rollup owner wallet with 10 ETH")
	SendValueTx(t, l1ChainID, l1GethClient, l1FundingWallet, l1RollupOwnerWallet.Address(), new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(10)), logger)
	// Send remaining balance of the l1RollupOwnerWallet afer
	defer func() {
		// get balance of l1RollupOwnerWallet
		logger.Info("Cleanup: Returning remaining funds from rollup owner wallet")
		remainingBalance, err := l1GethClient.BalanceAt(ctx, l1RollupOwnerWallet.Address(), nil)
		require.NoError(t, err)
		estimatedGas, _, gasFeeCap, err := CalculateGasParams(ctx, l1GethClient, l1RollupOwnerWallet.Address(), l1FundingWallet.Address(), big.NewInt(int64(0)), nil, logger)
		// Calculate the gas cost to subtract from the remaining balance
		gasCost := new(big.Int).Mul(big.NewInt(int64(estimatedGas)), gasFeeCap)
		// Subtract the gas cost from the remaining balance to avoid "insufficient funds" error
		balanceAfterGasCost := new(big.Int).Sub(remainingBalance, gasCost)
		require.NoError(t, err)
		SendValueTx(t, l1ChainID, l1GethClient, l1RollupOwnerWallet, l1FundingWallet.Address(), balanceAfterGasCost, logger)
	}()

	// Get the genesis config
	logger.Info("Getting L2 chain config")
	l2ChainConfig, err := l2Chain.Config()
	require.NoError(t, err)

	// Create fee checker
	logger.Info("Creating fee checker utility")
	feeChecker := NewFeeChecker(t, l2GethSeqClient, l2ChainConfig, logger)

	// Find gaspriceoracle contract
	logger.Info("Connecting to GasPriceOracle contract")
	gpoContract, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, l2GethSeqClient)
	require.NoError(t, err)

	logger.Info("Getting SystemConfig contract")
	systemConfigProxyAddr, ok := l2Chain.L1Addresses()["systemConfigProxy"]
	require.True(t, ok, "system config proxy address not found")
	systemConfig, err := bindings.NewSystemConfig(systemConfigProxyAddr, l1GethClient)
	require.NoError(t, err)

	owner, err := systemConfig.Owner(&bind.CallOpts{BlockNumber: nil})
	require.NoError(t, err)
	require.Equal(t, owner, l1RollupOwnerWallet.Address(), "system config proxy owner should be the rollup owner")

	// we take advantage here of the fact that the L2 dev wallets and the L1 wallets are the same. If they weren't we'd need to add another validator.
	logger.Info("Updating operator fee parameters",
		"constant", operatorFeeConstant,
		"scalar", operatorFeeScalar)
	_, receipt := UpdateOperatorFeeScalars(t, l1ChainID, l1GethClient, systemConfigProxyAddr, l1RollupOwnerWallet, operatorFeeConstant, operatorFeeScalar, logger)
	require.NoError(t, err)
	logger.Info("Operator fee parameters updated", "block", receipt.BlockNumber)

	// Verify the operator fee scalars were set correctly
	RequireOperatorFeeParamValues(t, systemConfig, receipt.BlockNumber, operatorFeeConstant, operatorFeeScalar)

	// sleep to allow for the L2 nodes to sync to L1 origin where operator fee was set
	logger.Info("Waiting 2 minutes for L2 nodes to sync with L1 origin where operator fee was set")
	time.Sleep(2 * time.Minute)

	// Check for chain fork
	startBlock, err := l2MultiClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	logger.Debug("Got L2 head block", "number", startBlock.Number)

	// Verify GPO isthmus view matches chain isthmus view
	gpoIsthmus, err := gpoContract.IsIsthmus(&bind.CallOpts{BlockNumber: startBlock.Number})
	require.NoError(t, err)
	require.True(t, gpoIsthmus, "GPO and chain must have same isthmus view")
	logger.Info("Verified GPO contract has correct Isthmus view")

	// Create balance reader
	logger.Info("Creating balance reader")
	balanceReader := NewBalanceReader(t, l2GethSeqClient, logger)

	// Get initial balances
	logger.Info("Sampling initial balances", "block", startBlock.Number.Uint64())
	startBalances := balanceReader.SampleBalances(ctx, startBlock.Number, l2FundingWallet.Address())
	logger.Debug("Initial balances", "balances", startBalances)

	// The amount transferred in the test transaction
	transferAmount := big.NewInt(params.Ether / 10) // 0.1 ETH
	logger.Info("Sending test transaction", "value", transferAmount)

	// Send the test transaction
	receipt, tx := SendValueTx(t, l2Chain.ID(), l2GethSeqClient, l2FundingWallet, common.Address{0xff, 0xff}, transferAmount, logger)
	logger.Info("Transaction confirmed",
		"block", receipt.BlockNumber.Uint64(),
		"hash", tx.Hash().Hex())

	// Get block header where transaction was included
	endHeader, err := l2MultiClient.HeaderByNumber(ctx, receipt.BlockNumber)
	require.NoError(t, err)
	require.Equal(t, endHeader.Coinbase, predeploys.SequencerFeeVaultAddr, "coinbase address should always be the same as the sequencer fee vault address")

	// Get final balances after transaction
	logger.Info("Sampling final balances", "block", endHeader.Number.Uint64())
	endBalances := balanceReader.SampleBalances(ctx, endHeader.Number, l2FundingWallet.Address())
	logger.Debug("Final balances", "balances", endBalances)

	// Calculate L1 fee for GPO verification
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err)
	l1Fee := feeChecker.L1Cost(tx.RollupCostData(), endHeader.Time)
	logger.Debug("Calculated L1 fee", "fee", l1Fee)

	// Verify gas price oracle L1 fee calculation
	adjustedGPOFee, err := gpoContract.GetL1Fee(&bind.CallOpts{BlockNumber: endHeader.Number}, txBytes)
	require.NoError(t, err)
	logger.Debug("GPO contract L1 fee", "fee", adjustedGPOFee)
	// Verify that GPO contract L1 fee calculation matches local L1 fee calculation
	require.Equal(t, l1Fee, adjustedGPOFee, "GPO reports L1 fee mismatch")
	// Verify execution L1 fee calculation matches GPO and local L1 fee calculation
	require.Equal(t, receipt.L1Fee, l1Fee, "l1 fee in receipt is correct")

	// Calculate expected fee changes from raw inputs
	logger.Info("Calculating expected balance changes based on transaction data")
	changes := feeChecker.CalculateExpectedBalanceChanges(
		receipt,
		endHeader,
		tx,
	)
	logger.Debug("Expected balance changes", "changes", changes)

	// Calculate expected end balances using the new method
	expectedEndBalances := startBalances.Add(changes)
	expectedEndBalances.BlockNumber = endHeader.Number
	logger.Debug("Expected final balances", "balances", expectedEndBalances)

	// Assert that actual end balances match what we calculated
	logger.Info("Verifying actual balances match expected balances")
	AssertSnapshotsEqual(t, expectedEndBalances, endBalances)
}

// operatorFeeTestScenario creates a test scenario for verifying fee calculations
func operatorFeeTestScenario(
	l1WalletGetter validators.WalletGetter,
	l2WalletGetter validators.WalletGetter,
	chainIdx uint64,
	logger log.Logger,
) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		logger.Info("Starting operator fee test scenario", "chain", chainIdx)
		// Get the low-level system and wallet
		l1Wallet := l1WalletGetter(t.Context())
		l2Wallet := l2WalletGetter(t.Context())
		logger.Info("Acquired wallets",
			"l1_wallet", l1Wallet.Address().Hex(),
			"l2_wallet", l2Wallet.Address().Hex())

		// Define test cases with different operator fee parameters
		testCases := []struct {
			name                   string
			operatorFeeConstant    uint64
			operatorFeeScalar      uint32
			expectedFeeCalculation string // Description of how fees should be calculated
		}{
			{
				name:                   "Zero fees",
				operatorFeeConstant:    0,
				operatorFeeScalar:      0,
				expectedFeeCalculation: "No operator fee should be charged",
			},
			{
				name:                   "Constant fee only",
				operatorFeeConstant:    1000,
				operatorFeeScalar:      0,
				expectedFeeCalculation: "Only constant fee component should be charged",
			},
			{
				name:                   "Scalar fee only",
				operatorFeeConstant:    0,
				operatorFeeScalar:      500, // 5% (scalar is in basis points)
				expectedFeeCalculation: "Fee should be proportional to base fee",
			},
			{
				name:                   "Both constant and scalar",
				operatorFeeConstant:    1000,
				operatorFeeScalar:      500, // 5%
				expectedFeeCalculation: "Fee should include both constant and proportional components",
			},
		}

		// For each test case, verify the operator fee parameters
		for _, tc := range testCases {
			t.Run(tc.name, func(t systest.T) {
				testLogger := logger.New("test_case", tc.name)
				testLogger.Info("Running test case",
					"description", tc.expectedFeeCalculation,
					"constant", tc.operatorFeeConstant,
					"scalar", tc.operatorFeeScalar)
				performOperatorFeeTest(t, sys, l1Wallet, l2Wallet, chainIdx, tc.operatorFeeConstant, tc.operatorFeeScalar, testLogger)
			})
		}
	}
}

func RequireOperatorFeeParamValues(t systest.T, systemConfig *bindings.SystemConfig, blockNumber *big.Int, expectedOperatorFeeConstant uint64, expectedOperatorFeeScalar uint32) {
	operatorFeeConstant, err := systemConfig.OperatorFeeConstant(&bind.CallOpts{BlockNumber: blockNumber})
	require.NoError(t, err)
	require.Equal(t, operatorFeeConstant, expectedOperatorFeeConstant, "operator fee constant should be 0")

	operatorFeeScalar, err := systemConfig.OperatorFeeScalar(&bind.CallOpts{BlockNumber: blockNumber})
	require.NoError(t, err)
	require.Equal(t, operatorFeeScalar, expectedOperatorFeeScalar, "operator fee scalar should be 0")
}

// BalanceReader provides methods to read balances from the chain
type BalanceReader struct {
	client *ethclient.Client
	t      systest.T
	logger log.Logger
}

// NewBalanceReader creates a new BalanceReader instance
func NewBalanceReader(t systest.T, client *ethclient.Client, logger log.Logger) *BalanceReader {
	return &BalanceReader{
		client: client,
		t:      t,
		logger: logger,
	}
}

// SampleBalances reads all the relevant balances at the given block number
// and returns a BalanceSnapshot containing the results
func (br *BalanceReader) SampleBalances(ctx context.Context, blockNumber *big.Int, walletAddr common.Address) *BalanceSnapshot {
	br.logger.Debug("Sampling balances",
		"block", blockNumber,
		"wallet", walletAddr.Hex())
	// Get the block to determine coinbase
	var header *gethTypes.Header
	var err error

	if blockNumber == nil {
		header, err = br.client.HeaderByNumber(ctx, nil)
	} else {
		header, err = br.client.HeaderByNumber(ctx, blockNumber)
	}
	require.NoError(br.t, err)

	// Use the block number from the header if none was provided
	if blockNumber == nil {
		blockNumber = header.Number
		br.logger.Debug("No block number provided, using current block", "block", blockNumber)
	}

	coinbaseAddr := header.Coinbase
	br.logger.Debug("Block coinbase", "address", coinbaseAddr.Hex())

	// Read all balances
	baseFeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.BaseFeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	l1FeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.L1FeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	sequencerFeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.SequencerFeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	operatorFeeVaultBalance, err := br.client.BalanceAt(ctx, predeploys.OperatorFeeVaultAddr, blockNumber)
	require.NoError(br.t, err)

	walletBalance, err := br.client.BalanceAt(ctx, walletAddr, blockNumber)
	require.NoError(br.t, err)

	br.logger.Debug("Sampled balances",
		"baseFee", baseFeeVaultBalance,
		"l1Fee", l1FeeVaultBalance,
		"sequencerFee", sequencerFeeVaultBalance,
		"operatorFee", operatorFeeVaultBalance,
		"wallet", walletBalance)

	return &BalanceSnapshot{
		BlockNumber:         blockNumber,
		BaseFeeVaultBalance: baseFeeVaultBalance,
		L1FeeVaultBalance:   l1FeeVaultBalance,
		SequencerFeeVault:   sequencerFeeVaultBalance,
		OperatorFeeVault:    operatorFeeVaultBalance,
		FromBalance:         walletBalance,
	}
}

func SendValueTx(t systest.T, chainID *big.Int, client *ethclient.Client, from system.Wallet, to common.Address, value *big.Int, logger log.Logger) (*gethTypes.Receipt, *gethTypes.Transaction) {
	ctx := t.Context()
	logger.Info("Sending value transaction",
		"from", from.Address().Hex(),
		"to", to.Hex(),
		"value", value)

	// Get pending nonce
	nonce, err := client.PendingNonceAt(ctx, from.Address())
	require.NoError(t, err)
	logger.Debug("Using nonce", "nonce", nonce)

	// Calculate gas parameters using utility function
	gasLimit, gasTipCap, gasFeeCap, err := CalculateGasParams(ctx, client, from.Address(), to, value, nil, logger)
	require.NoError(t, err, "Failed to calculate gas parameters")

	// Create transaction
	txData := &gethTypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     value,
		Data:      nil,
	}

	// Sign transaction
	tx := gethTypes.NewTx(txData)
	signedTx, err := gethTypes.SignTx(tx, gethTypes.LatestSignerForChainID(chainID), from.PrivateKey())
	require.NoError(t, err)
	logger.Debug("Transaction signed", "hash", signedTx.Hash().Hex())

	// Send transaction
	logger.Info("Sending transaction to the network")
	err = client.SendTransaction(ctx, signedTx)
	require.NoError(t, err)

	// Wait for transaction receipt with timeout
	logger.Info("Waiting for transaction confirmation")
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	receipt, err := waitForTransaction(ctx, client, signedTx.Hash())
	require.NoError(t, err, "Failed to wait for transaction receipt")
	require.NotNil(t, receipt)
	require.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
	logger.Info("Transaction confirmed",
		"block", receipt.BlockNumber,
		"gasUsed", receipt.GasUsed)

	return receipt, tx
}

// CalculateGasParams calculates appropriate gas parameters for a transaction
func CalculateGasParams(ctx context.Context, client *ethclient.Client, from common.Address, to common.Address, value *big.Int, data []byte, logger log.Logger) (estimatedGas uint64, gasTipCap *big.Int, gasFeeCap *big.Int, err error) {
	// Get current block header for base fee
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to get header: %w", err)
	}
	logger.Debug("Current block", "number", header.Number, "baseFee", header.BaseFee)

	// Get suggested gas tip
	gasTipCap, err = client.SuggestGasTipCap(ctx)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to get suggested gas tip: %w", err)
	}
	logger.Debug("Using suggested gas tip", "tip", gasTipCap)

	// Calculate gas fee cap (2 * baseFee + tip)
	gasFeeCap = new(big.Int).Add(
		new(big.Int).Mul(header.BaseFee, big.NewInt(2)),
		gasTipCap,
	)
	logger.Debug("Calculated gas fee cap", "feeCap", gasFeeCap)

	// Estimate gas limit
	estimatedGas, err = client.EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    &to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		logger.Warn("Gas estimation error", "error", err)
		// Return error but also provide a fallback estimated gas in case caller wants to continue
		return 300000, gasTipCap, gasFeeCap, fmt.Errorf("failed to estimate gas: %w", err)
	}
	logger.Debug("Estimated gas limit", "limit", estimatedGas)

	return estimatedGas, gasTipCap, gasFeeCap, nil
}

func UpdateOperatorFeeScalars(t systest.T, l1ChainID *big.Int, client *ethclient.Client, systemConfigAddress common.Address, wallet system.Wallet, operatorFeeConstant uint64, operatorFeeScalar uint32, logger log.Logger) (*gethTypes.Transaction, *gethTypes.Receipt) {
	ctx := t.Context()
	logger.Info("Updating operator fee scalars",
		"constant", operatorFeeConstant,
		"scalar", operatorFeeScalar)

	nonce, err := client.PendingNonceAt(ctx, wallet.Address())
	require.NoError(t, err)
	logger.Debug("Using nonce",
		"nonce", nonce,
		"wallet", wallet.Address().Hex())

	// Construct call input
	logger.Debug("Constructing function call to setOperatorFeeScalars")
	funcSetOperatorFeeScalars := w3.MustNewFunc(`setOperatorFeeScalars(uint32 _operatorFeeScalar, uint64 _operatorFeeConstant)`, "")
	args, err := funcSetOperatorFeeScalars.EncodeArgs(
		operatorFeeScalar,
		operatorFeeConstant,
	)
	require.NoError(t, err)

	// Calculate gas parameters
	gasLimit, gasTipCap, gasFeeCap, err := CalculateGasParams(ctx, client, wallet.Address(), systemConfigAddress, big.NewInt(0), args, logger)
	if err != nil {
		logger.Warn("Error calculating gas parameters", "error", err)
	}

	tx := gethTypes.NewTx(&gethTypes.DynamicFeeTx{
		To:        &systemConfigAddress,
		Gas:       gasLimit,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Nonce:     nonce,
		Value:     big.NewInt(0),
		Data:      args,
	})
	signer := gethTypes.NewLondonSigner(l1ChainID)
	signedTx, err := gethTypes.SignTx(tx, signer, wallet.PrivateKey())
	require.NoError(t, err)
	logger.Debug("Transaction signed", "hash", signedTx.Hash().Hex())

	logger.Info("Sending transaction to the network")
	err = client.SendTransaction(context.Background(), signedTx)
	require.NoError(t, err)

	// Wait for transaction receipt with timeout
	logger.Info("Waiting for transaction confirmation")
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	receipt, err := waitForTransaction(ctx, client, signedTx.Hash())
	require.NoError(t, err, "Failed to wait for transaction receipt")
	require.NotNil(t, receipt)
	require.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
	logger.Info("Transaction confirmed",
		"block", receipt.BlockNumber,
		"gasUsed", receipt.GasUsed)

	return tx, receipt
}
