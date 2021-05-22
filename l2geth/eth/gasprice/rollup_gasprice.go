package gasprice

import (
	"context"
	"math/big"
	"sync"
)

type RollupOracle struct {
	dataPrice          *big.Int
	executionPrice     *big.Int
	dataPriceLock      sync.RWMutex
	executionPriceLock sync.RWMutex
}

func NewRollupOracle(dataPrice *big.Int, executionPrice *big.Int) *RollupOracle {
	return &RollupOracle{
		dataPrice:      dataPrice,
		executionPrice: executionPrice,
	}
}

/// SuggestDataPrice returns the gas price which should be charged per byte of published
/// data by the sequencer.
func (gpo *RollupOracle) SuggestDataPrice(ctx context.Context) (*big.Int, error) {
	gpo.dataPriceLock.RLock()
	price := gpo.dataPrice
	gpo.dataPriceLock.RUnlock()
	return price, nil
}

func (gpo *RollupOracle) SetDataPrice(dataPrice *big.Int) {
	gpo.dataPriceLock.Lock()
	gpo.dataPrice = dataPrice
	gpo.dataPriceLock.Unlock()
}

/// SuggestExecutionPrice returns the gas price which should be charged per unit of gas
/// set manually by the sequencer depending on congestion
func (gpo *RollupOracle) SuggestExecutionPrice(ctx context.Context) (*big.Int, error) {
	gpo.executionPriceLock.RLock()
	price := gpo.executionPrice
	gpo.executionPriceLock.RUnlock()
	return price, nil
}

func (gpo *RollupOracle) SetExecutionPrice(executionPrice *big.Int) {
	gpo.executionPriceLock.Lock()
	gpo.executionPrice = executionPrice
	gpo.executionPriceLock.Unlock()
}
