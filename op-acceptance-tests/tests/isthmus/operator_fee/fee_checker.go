package isthmus

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

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
	l1Fee := fc.L1Cost(tx.RollupCostData(), header.Time)

	// Calculate operator fee
	fc.logger.Debug("Calculating operator fee", "gasUsed", gasUsedUint64, "blockTime", header.Time)
	operatorFee := fc.operatorFeeFn(gasUsedUint64, header.Time).ToBig()

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
