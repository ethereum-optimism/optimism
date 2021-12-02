package gasprices

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

type GetLatestBlockNumberFn func() (uint64, error)
type UpdateL2GasPriceFn func(uint64) error
type GetGasUsedByBlockFn func(*big.Int) (uint64, error)

type GasPriceUpdater struct {
	mu                     *sync.RWMutex
	gasPricer              *GasPricer
	epochStartBlockNumber  uint64
	averageBlockGasLimit   uint64
	epochLengthSeconds     uint64
	getLatestBlockNumberFn GetLatestBlockNumberFn
	getGasUsedByBlockFn    GetGasUsedByBlockFn
	updateL2GasPriceFn     UpdateL2GasPriceFn
}

func NewGasPriceUpdater(
	gasPricer *GasPricer,
	epochStartBlockNumber uint64,
	averageBlockGasLimit uint64,
	epochLengthSeconds uint64,
	getLatestBlockNumberFn GetLatestBlockNumberFn,
	getGasUsedByBlockFn GetGasUsedByBlockFn,
	updateL2GasPriceFn UpdateL2GasPriceFn,
) (*GasPriceUpdater, error) {
	if averageBlockGasLimit < 1 {
		return nil, errors.New("averageBlockGasLimit cannot be less than 1 gas")
	}
	if epochLengthSeconds < 1 {
		return nil, errors.New("epochLengthSeconds cannot be less than 1 second")
	}
	return &GasPriceUpdater{
		mu:                     new(sync.RWMutex),
		gasPricer:              gasPricer,
		epochStartBlockNumber:  epochStartBlockNumber,
		epochLengthSeconds:     epochLengthSeconds,
		averageBlockGasLimit:   averageBlockGasLimit,
		getLatestBlockNumberFn: getLatestBlockNumberFn,
		getGasUsedByBlockFn:    getGasUsedByBlockFn,
		updateL2GasPriceFn:     updateL2GasPriceFn,
	}, nil
}

func (g *GasPriceUpdater) UpdateGasPrice() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	latestBlockNumber, err := g.getLatestBlockNumberFn()
	if err != nil {
		return err
	}
	if latestBlockNumber < g.epochStartBlockNumber {
		return errors.New("Latest block number less than the last epoch's block number")
	}

	if latestBlockNumber == g.epochStartBlockNumber {
		log.Debug("latest block number is equal to epoch start block number", "number", latestBlockNumber)
		return nil
	}

	// Accumulate the amount of gas that has been used in the epoch
	totalGasUsed := uint64(0)
	for i := g.epochStartBlockNumber + 1; i <= latestBlockNumber; i++ {
		gasUsed, err := g.getGasUsedByBlockFn(new(big.Int).SetUint64(i))
		log.Trace("fetching gas used", "height", i, "gas-used", gasUsed, "total-gas", totalGasUsed)
		if err != nil {
			return err
		}
		totalGasUsed += gasUsed
	}

	averageGasPerSecond := float64(totalGasUsed) / float64(g.epochLengthSeconds)

	log.Debug("UpdateGasPrice", "average-gas-per-second", averageGasPerSecond, "current-price", g.gasPricer.curPrice)
	_, err = g.gasPricer.CompleteEpoch(averageGasPerSecond)
	if err != nil {
		return err
	}
	g.epochStartBlockNumber = latestBlockNumber
	err = g.updateL2GasPriceFn(g.gasPricer.curPrice)
	if err != nil {
		return err
	}
	return nil
}

func (g *GasPriceUpdater) GetGasPrice() uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.gasPricer.curPrice
}
