package txinclude

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core/types"
)

// TxBudget provides budgeting helpers oriented around a transaction's lifecycle.
type TxBudget struct {
	inner BasicBudget
	cfg   *txBudgetConfig
}

type TxBudgetOption func(*txBudgetConfig)

type txBudgetConfig struct {
	oracle OPCostOracle
}

func WithOPCostOracle(oracle OPCostOracle) TxBudgetOption {
	return func(cfg *txBudgetConfig) {
		cfg.oracle = oracle
	}
}

func NewTxBudget(inner BasicBudget, opts ...TxBudgetOption) *TxBudget {
	cfg := &txBudgetConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &TxBudget{
		inner: inner,
		cfg:   cfg,
	}
}

var _ Budget = (*TxBudget)(nil)

// BeforeResubmit calculates the cost of tx. If the new cost is greather than oldCost, it debits
// the difference. If the new cost is less than oldCost, it credits the difference.
func (b *TxBudget) BeforeResubmit(oldCost eth.ETH, tx *types.Transaction) (eth.ETH, error) {
	// Gas cost.
	total := new(big.Int).Mul(tx.GasFeeCap(), new(big.Int).SetUint64(tx.Gas()))
	if tx.Type() == types.BlobTxType {
		// Blob gas cost.
		blobGasFee := tx.BlobGasFeeCap()
		total.Add(total, blobGasFee.Mul(blobGasFee, new(big.Int).SetUint64(tx.BlobGas())))
	}
	// OP cost.
	if b.cfg.oracle != nil {
		opCost := b.cfg.oracle.OPCost(tx)
		total.Add(total, opCost)
	}

	newCost := eth.WeiBig(total)
	if newCost.Gt(oldCost) {
		if _, err := b.inner.Debit(newCost.Sub(oldCost)); err != nil {
			return eth.ETH{}, err
		}
	} else if newCost.Lt(oldCost) {
		b.inner.Credit(oldCost.Sub(newCost))
	}
	return newCost, nil
}

// AfterCancel credits cost.
func (b *TxBudget) AfterCancel(cost eth.ETH, _ *types.Transaction) {
	b.inner.Credit(cost)
}

// AfterIncluded credits the difference between the budgeted cost and the actual cost. It is
// assumed that the budgeted cost is always greater than the actual cost.
func (b *TxBudget) AfterIncluded(budgetedCost eth.ETH, tx *IncludedTx) {
	// gasCost
	receipt := tx.Receipt
	gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
	actualCost := gasUsed.Mul(gasUsed, receipt.EffectiveGasPrice)
	if receipt.Type == types.BlobTxType {
		// blobGasCost
		blobCost := new(big.Int).SetUint64(receipt.BlobGasUsed)
		blobCost.Mul(blobCost, receipt.BlobGasPrice)
		actualCost.Add(actualCost, blobCost)
	}

	// l1Cost
	if receipt.L1BaseFeeScalar != nil {
		l1BaseFeeScalar := new(big.Int).SetUint64(*receipt.L1BaseFeeScalar)
		l1BlobBaseFeeScalar := new(big.Int).SetUint64(*receipt.L1BlobBaseFeeScalar)
		costFunc := types.NewL1CostFuncFjord(receipt.L1GasPrice, receipt.L1BlobBaseFee, l1BaseFeeScalar, l1BlobBaseFeeScalar)
		l1Cost, _ := costFunc(tx.Transaction.RollupCostData())
		actualCost.Add(actualCost, l1Cost)
	}

	// operatorCost
	if receipt.OperatorFeeScalar != nil {
		// https://github.com/ethereum-optimism/op-geth/blob/6005dd53e1b50fe5a3f59764e3e2056a639eff2f/core/types/rollup_cost.go#L244-L247
		// Also see: https://specs.optimism.io/protocol/isthmus/exec-engine.html#operator-operatorCost
		operatorCost := new(big.Int).SetUint64(receipt.GasUsed)
		operatorCost.Mul(operatorCost, new(big.Int).SetUint64(*receipt.OperatorFeeScalar))
		operatorCost = operatorCost.Div(operatorCost, oneMillion)
		operatorCost = operatorCost.Add(operatorCost, new(big.Int).SetUint64(*receipt.OperatorFeeConstant))
		actualCost.Add(actualCost, operatorCost)
	}

	b.inner.Credit(budgetedCost.Sub(eth.WeiBig(actualCost)))
}
