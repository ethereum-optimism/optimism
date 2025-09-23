package operatorfee

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type stateGetterAdapterFactory struct {
	t      devtest.T
	client *ethclient.Client
}

// stateGetterAdapter adapts the ethclient to implement the StateGetter interface
type stateGetterAdapter struct {
	t           devtest.T
	client      *ethclient.Client
	ctx         context.Context
	blockNumber *big.Int
}

func (f *stateGetterAdapterFactory) NewStateGetterAdapter(blockNumber *big.Int) *stateGetterAdapter {
	return &stateGetterAdapter{
		t:           f.t,
		client:      f.client,
		ctx:         f.t.Ctx(),
		blockNumber: blockNumber,
	}
}

// GetState implements the StateGetter interface
func (sga *stateGetterAdapter) GetState(addr common.Address, key common.Hash) common.Hash {
	var result common.Hash
	val, err := sga.client.StorageAt(sga.ctx, addr, key, sga.blockNumber)
	require.NoError(sga.t, err)
	copy(result[:], val)
	return result
}

// FeeChecker provides methods to calculate various types of fees
type FeeChecker struct {
	config     *params.ChainConfig
	logger     log.Logger
	sgaFactory *stateGetterAdapterFactory
}

// NewFeeChecker creates a new FeeChecker instance
func NewFeeChecker(t devtest.T, client *ethclient.Client, chainConfig *params.ChainConfig, logger log.Logger) *FeeChecker {
	logger.Debug("Creating fee checker", "chainID", chainConfig.ChainID)

	// Create state getter adapter factory
	sgaFactory := &stateGetterAdapterFactory{
		t:      t,
		client: client,
	}

	return &FeeChecker{
		config:     chainConfig,
		sgaFactory: sgaFactory,
		logger:     logger,
	}
}

// L1Cost calculates the L1 fee for a transaction
func (fc *FeeChecker) L1Cost(rcd gethTypes.RollupCostData, blockTime uint64, blockNumber *big.Int) *big.Int {

	// Create L1 cost function
	l1CostFn := gethTypes.NewL1CostFunc(fc.config, fc.sgaFactory.NewStateGetterAdapter(blockNumber))
	return l1CostFn(rcd, blockTime)
}

// OperatorFee calculates the operator fee for a transaction
func (fc *FeeChecker) OperatorFee(gasUsed uint64, blockTime uint64, blockNumber *big.Int) *big.Int {
	operatorFeeFn := gethTypes.NewOperatorCostFunc(fc.config, fc.sgaFactory.NewStateGetterAdapter(blockNumber))
	return operatorFeeFn(gasUsed, blockTime).ToBig()
}

// CalculateExpectedBalanceChanges creates a BalanceSnapshot containing expected fee movements
// Calculates all fees internally from raw inputs
func (fc *FeeChecker) CalculateExpectedBalanceChanges(
	gasUsedUint64 uint64,
	header *gethTypes.Header,
	tx *gethTypes.Transaction,
) *BalanceSnapshot {
	// Convert the gas used (uint64) to a big.Int.
	gasUsed := new(big.Int).SetUint64(gasUsedUint64)

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

	fc.logger.Debug("Calculating L1 fee", "rollupCostData", tx.RollupCostData(), "blockTime", header.Time)
	l1Fee := fc.L1Cost(tx.RollupCostData(), header.Time, header.Number)

	// Calculate operator fee
	fc.logger.Debug("Calculating operator fee", "gasUsed", gasUsedUint64, "blockTime", header.Time)
	operatorFee := fc.OperatorFee(gasUsedUint64, header.Time, header.Number)

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
