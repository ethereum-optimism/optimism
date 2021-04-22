package gasprice

import (
	"context"
	"math/big"
)

type RollupOracle struct {
	dataPrice      *big.Int
	executionPrice *big.Int
}

func NewRollupOracle(dataPrice *big.Int, executionPrice *big.Int) *RollupOracle {
	return &RollupOracle{dataPrice, executionPrice}
}

/// SuggestDataPrice returns the gas price which should be charged per byte of published
/// data by the sequencer.
func (gpo *RollupOracle) SuggestDataPrice(ctx context.Context) (*big.Int, error) {
	return gpo.dataPrice, nil
}

func (gpo *RollupOracle) SetDataPrice(dataPrice *big.Int) {
	gpo.dataPrice = dataPrice
}

/// SuggestExecutionPrice returns the gas price which should be charged per unit of gas
/// set manually by the sequencer depending on congestion
func (gpo *RollupOracle) SuggestExecutionPrice(ctx context.Context) (*big.Int, error) {
	return gpo.executionPrice, nil
}

func (gpo *RollupOracle) SetExecutionPrice(executionPrice *big.Int) {
	gpo.executionPrice = executionPrice
}
