package gasprices

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

type GetLatestBlockNumberFn func() (uint64, error)
type UpdateL2GasPriceFn func(uint64) error

type GasPriceUpdater struct {
	mu                     *sync.RWMutex
	gasPricer              *GasPricer
	epochStartBlockNumber  uint64
	averageBlockGasLimit   float64
	epochLengthSeconds     uint64
	getLatestBlockNumberFn GetLatestBlockNumberFn
	updateL2GasPriceFn     UpdateL2GasPriceFn
}

func GetAverageGasPerSecond(
	epochStartBlockNumber uint64,
	latestBlockNumber uint64,
	epochLengthSeconds uint64,
	averageBlockGasLimit uint64,
) float64 {
	blocksPassed := latestBlockNumber - epochStartBlockNumber
	return float64(blocksPassed * averageBlockGasLimit / epochLengthSeconds)
}

func NewGasPriceUpdater(
	gasPricer *GasPricer,
	epochStartBlockNumber uint64,
	averageBlockGasLimit float64,
	epochLengthSeconds uint64,
	getLatestBlockNumberFn GetLatestBlockNumberFn,
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
	if latestBlockNumber < uint64(g.epochStartBlockNumber) {
		return errors.New("Latest block number less than the last epoch's block number")
	}
	averageGasPerSecond := GetAverageGasPerSecond(
		g.epochStartBlockNumber,
		latestBlockNumber,
		uint64(g.epochLengthSeconds),
		uint64(g.averageBlockGasLimit),
	)
	log.Debug("UpdateGasPrice", "averageGasPerSecond", averageGasPerSecond, "current-price", g.gasPricer.curPrice)
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
