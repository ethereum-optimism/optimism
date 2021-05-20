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

func (gpo *RollupOracle) SetDataPrice(dataPrice *big.Int) {
	gpo.dataPriceLock.Lock()
	defer gpo.dataPriceLock.Unlock()
	price := core.RoundL1GasPrice(gpo.dataPrice.Uint64())
	gpo.dataPrice = gpo.dataPrice.SetUint64(price)
	log.Info("Set L1 Gas Price", "gasprice", gpo.dataPrice)
}

/// SuggestExecutionPrice returns the gas price which should be charged per unit of gas
/// set manually by the sequencer depending on congestion
func (gpo *RollupOracle) SuggestExecutionPrice(ctx context.Context) (*big.Int, error) {
	gpo.executionPriceLock.RLock()
	defer gpo.executionPriceLock.RUnlock()
	return gpo.executionPrice, nil
}

func (gpo *RollupOracle) SetExecutionPrice(executionPrice *big.Int) {
	gpo.executionPriceLock.Lock()
	defer gpo.executionPriceLock.Unlock()
	price := core.RoundL2GasPrice(executionPrice.Uint64())
	gpo.executionPrice = gpo.executionPrice.SetUint64(price)
	log.Info("Set L2 Gas Price", "gasprice", gpo.executionPrice)
}
