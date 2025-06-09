package txinclude

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core/types"
)

// txBudget provides budgeting helpers oriented around a transaction's lifecycle.
type txBudget struct {
	inner Budget
}

func newTxBudget(inner Budget) *txBudget {
	return &txBudget{
		inner: inner,
	}
}

func (b *txBudget) resubmitting(oldCost, newCost eth.ETH) error {
	if newCost.Gt(oldCost) {
		if err := b.inner.Debit(newCost.Sub(oldCost)); err != nil {
			return err
		}
	} else if newCost.Lt(oldCost) {
		b.inner.Credit(oldCost.Sub(newCost))
	}
	return nil
}

func (b *txBudget) canceling(cost eth.ETH) {
	b.inner.Credit(cost)
}

func (b *txBudget) included(tx *IncludedTx) {
	actualCost := new(big.Int).SetUint64(tx.Receipt.GasUsed)
	actualCost.Mul(actualCost, tx.Receipt.EffectiveGasPrice)
	if tx.Receipt.Type == types.BlobTxType {
		blobCost := new(big.Int).SetUint64(tx.Receipt.BlobGasUsed)
		blobCost.Mul(blobCost, tx.Receipt.BlobGasPrice)
		actualCost.Add(actualCost, blobCost)
	}
	budgetedCost := maxGasCost(tx.Transaction)
	b.inner.Credit(budgetedCost.Sub(eth.WeiBig(actualCost)))
}

func maxGasCost(tx *types.Transaction) eth.ETH {
	// See the implementation of tx.Cost().
	// We do the same calculation without adding tx.Value().
	gasPrice := tx.GasFeeCap()
	total := gasPrice.Mul(gasPrice, new(big.Int).SetUint64(tx.Gas()))
	if tx.Type() == types.BlobTxType {
		blobGasPrice := tx.BlobGasFeeCap()
		total.Add(total, blobGasPrice.Mul(blobGasPrice, new(big.Int).SetUint64(tx.BlobGas())))
	}
	return eth.WeiBig(total)
}
