package gasprice

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// RollupOracle holds the L1 and L2 gas prices for fee calculation
type RollupOracle struct {
	l1GasPrice     *big.Int
	l2GasPrice     *big.Int
	l1GasPriceLock sync.RWMutex
	l2GasPriceLock sync.RWMutex
}

// NewRollupOracle returns an initialized RollupOracle
func NewRollupOracle() *RollupOracle {
	return &RollupOracle{
		l1GasPrice:     new(big.Int),
		l2GasPrice:     new(big.Int),
		l1GasPriceLock: sync.RWMutex{},
		l2GasPriceLock: sync.RWMutex{},
	}
}

// SuggestL1GasPrice returns the gas price which should be charged per byte of published
// data by the sequencer.
func (gpo *RollupOracle) SuggestL1GasPrice(ctx context.Context) (*big.Int, error) {
	gpo.l1GasPriceLock.RLock()
	defer gpo.l1GasPriceLock.RUnlock()
	return gpo.l1GasPrice, nil
}

// SetL1GasPrice returns the current L1 gas price
func (gpo *RollupOracle) SetL1GasPrice(gasPrice *big.Int) error {
	gpo.l1GasPriceLock.Lock()
	defer gpo.l1GasPriceLock.Unlock()
	gpo.l1GasPrice = gasPrice
	log.Info("Set L1 Gas Price", "gasprice", gpo.l1GasPrice)
	return nil
}

// SuggestL2GasPrice returns the gas price which should be charged per unit of gas
// set manually by the sequencer depending on congestion
func (gpo *RollupOracle) SuggestL2GasPrice(ctx context.Context) (*big.Int, error) {
	gpo.l2GasPriceLock.RLock()
	defer gpo.l2GasPriceLock.RUnlock()
	return gpo.l2GasPrice, nil
}

// SetL2GasPrice returns the current L2 gas price
func (gpo *RollupOracle) SetL2GasPrice(gasPrice *big.Int) error {
	gpo.l2GasPriceLock.Lock()
	defer gpo.l2GasPriceLock.Unlock()
	gpo.l2GasPrice = gasPrice
	log.Info("Set L2 Gas Price", "gasprice", gpo.l2GasPrice)
	return nil
}
