package gasprice

import (
	"context"
	"math/big"
)

type L1Oracle struct {
	gasPrice *big.Int
}

func NewL1Oracle(gasPrice *big.Int) *L1Oracle {
	return &L1Oracle{gasPrice}
}

/// SuggestDataPrice returns the gas price which should be charged per byte of published
/// data by the sequencer.
func (gpo *L1Oracle) SuggestDataPrice(ctx context.Context) (*big.Int, error) {
	return gpo.gasPrice, nil
}

func (gpo *L1Oracle) SetL1GasPrice(gasPrice *big.Int) {
	gpo.gasPrice = gasPrice
}
