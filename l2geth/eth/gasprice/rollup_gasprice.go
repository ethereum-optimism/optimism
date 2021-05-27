package gasprice

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rollup/fees"
)

// RollupOracle holds the L1 and L2 gas prices for fee calculation
type RollupOracle struct {
	dataPrice          *big.Int
	executionPrice     *big.Int
	dataPriceLock      sync.RWMutex
	executionPriceLock sync.RWMutex
}

// NewRollupOracle returns an initialized RollupOracle
func NewRollupOracle(dataPrice *big.Int, executionPrice *big.Int) *RollupOracle {
	return &RollupOracle{
		dataPrice:      dataPrice,
		executionPrice: executionPrice,
	}
}

// SuggestL1GasPrice returns the gas price which should be charged per byte of published
// data by the sequencer.
func (gpo *RollupOracle) SuggestL1GasPrice(ctx context.Context) (*big.Int, error) {
	gpo.dataPriceLock.RLock()
	defer gpo.dataPriceLock.RUnlock()
	return gpo.dataPrice, nil
}

// SetL1GasPrice returns the current L1 gas price
func (gpo *RollupOracle) SetL1GasPrice(gasPrice *big.Int) error {
	gpo.dataPriceLock.Lock()
	defer gpo.dataPriceLock.Unlock()
	if err := fees.VerifyGasPrice(gasPrice); err != nil {
		return err
	}
	gpo.dataPrice = gasPrice
	log.Info("Set L1 Gas Price", "gasprice", gpo.dataPrice)
	return nil
}

// SuggestL2GasPrice returns the gas price which should be charged per unit of gas
// set manually by the sequencer depending on congestion
func (gpo *RollupOracle) SuggestL2GasPrice(ctx context.Context) (*big.Int, error) {
	gpo.executionPriceLock.RLock()
	defer gpo.executionPriceLock.RUnlock()
	return gpo.executionPrice, nil
}

// SetL2GasPrice returns the current L2 gas price
func (gpo *RollupOracle) SetL2GasPrice(gasPrice *big.Int) error {
	gpo.executionPriceLock.Lock()
	defer gpo.executionPriceLock.Unlock()
	if err := fees.VerifyGasPrice(gasPrice); err != nil {
		return err
	}
	gpo.executionPrice = gasPrice
	log.Info("Set L2 Gas Price", "gasprice", gpo.executionPrice)
	return nil
}
