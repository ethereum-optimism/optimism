package gasprice

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/l2geth/log"
	"github.com/ethereum-optimism/optimism/l2geth/rollup/fees"
)

// RollupOracle holds the L1 and L2 gas prices for fee calculation
type RollupOracle struct {
	l1GasPrice     *big.Int
	l2GasPrice     *big.Int
	overhead       *big.Int
	scalar         *big.Float
	l1GasPriceLock sync.RWMutex
	l2GasPriceLock sync.RWMutex
	overheadLock   sync.RWMutex
	scalarLock     sync.RWMutex
}

// NewRollupOracle returns an initialized RollupOracle
func NewRollupOracle() *RollupOracle {
	return &RollupOracle{
		l1GasPrice: new(big.Int),
		l2GasPrice: new(big.Int),
		overhead:   new(big.Int),
		scalar:     new(big.Float),
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

// SuggestOverhead returns the cached overhead value from the
// OVM_GasPriceOracle
func (gpo *RollupOracle) SuggestOverhead(ctx context.Context) (*big.Int, error) {
	gpo.overheadLock.RLock()
	defer gpo.overheadLock.RUnlock()
	return gpo.overhead, nil
}

// SetOverhead caches the overhead value that is set in the
// OVM_GasPriceOracle
func (gpo *RollupOracle) SetOverhead(overhead *big.Int) error {
	gpo.overheadLock.Lock()
	defer gpo.overheadLock.Unlock()
	gpo.overhead = overhead
	log.Info("Set batch overhead", "overhead", overhead)
	return nil
}

// SuggestScalar returns the cached scalar value
func (gpo *RollupOracle) SuggestScalar(ctx context.Context) (*big.Float, error) {
	gpo.scalarLock.RLock()
	defer gpo.scalarLock.RUnlock()
	return gpo.scalar, nil
}

// SetScalar sets the scalar value held in the OVM_GasPriceOracle
func (gpo *RollupOracle) SetScalar(scalar *big.Int, decimals *big.Int) error {
	gpo.scalarLock.Lock()
	defer gpo.scalarLock.Unlock()
	value := fees.ScaleDecimals(scalar, decimals)
	gpo.scalar = value
	log.Info("Set scalar", "scalar", gpo.scalar)
	return nil
}
