package gasprice

import (
	"context"
	"math/big"
)

type RollupOracle struct {
	dataPrice *big.Int
}

func NewRollupOracle(dataPrice *big.Int) *RollupOracle {
	return &RollupOracle{dataPrice}
}

/// SuggestDataPrice returns the gas price which should be charged per byte of published
/// data by the sequencer.
func (gpo *RollupOracle) SuggestDataPrice(ctx context.Context) (*big.Int, error) {
	return gpo.dataPrice, nil
}

func (gpo *RollupOracle) SetDataPrice(dataPrice *big.Int) {
	gpo.dataPrice = dataPrice
}
