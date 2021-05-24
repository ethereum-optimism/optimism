package gasprice

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
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
	defer gpo.dataPriceLock.RUnlock()
	return gpo.dataPrice, nil
}

func (gpo *RollupOracle) SetDataPrice(dataPrice *big.Int) error {
	gpo.dataPriceLock.Lock()
	defer gpo.dataPriceLock.Unlock()
	if err := core.VerifyL1GasPrice(dataPrice); err != nil {
		return err
	}
	gpo.dataPrice = dataPrice
	log.Info("Set L1 Gas Price", "gasprice", gpo.dataPrice)
	return nil
}

/// SuggestExecutionPrice returns the gas price which should be charged per unit of gas
/// set manually by the sequencer depending on congestion
func (gpo *RollupOracle) SuggestExecutionPrice(ctx context.Context) (*big.Int, error) {
	gpo.executionPriceLock.RLock()
	defer gpo.executionPriceLock.RUnlock()
	return gpo.executionPrice, nil
}

func (gpo *RollupOracle) SetExecutionPrice(executionPrice *big.Int) error {
	gpo.executionPriceLock.Lock()
	defer gpo.executionPriceLock.Unlock()
	if err := core.VerifyL2GasPrice(executionPrice); err != nil {
		return err
	}
	gpo.executionPrice = executionPrice
	log.Info("Set L2 Gas Price", "gasprice", gpo.executionPrice)
	return nil
}
