package gasprices

import "sync"

type GetLatestBlockNumberFn func() (uint64, error)
type UpdateL2GasPriceFn func(float64) error

type GasPriceUpdater struct {
	mu                     *sync.RWMutex
	gasPricer              *L2GasPricer
	epochStartBlockNumber  float64
	averageBlockGasLimit   float64
	getLatestBlockNumberFn GetLatestBlockNumberFn
	updateL2GasPriceFn     UpdateL2GasPriceFn
}

func NewGasPriceUpdater(
	gasPricer *L2GasPricer,
	epochStartBlockNumber float64,
	averageBlockGasLimit float64,
	getLatestBlockNumberFn GetLatestBlockNumberFn,
	updateL2GasPriceFn UpdateL2GasPriceFn,
) *GasPriceUpdater {
	return &GasPriceUpdater{
		gasPricer:              gasPricer,
		epochStartBlockNumber:  epochStartBlockNumber,
		averageBlockGasLimit:   averageBlockGasLimit,
		getLatestBlockNumberFn: getLatestBlockNumberFn,
		updateL2GasPriceFn:     updateL2GasPriceFn,
	}
}

func (g *GasPriceUpdater) UpdateGasPrice() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	latestBlockNumber, err := g.getLatestBlockNumberFn()
	if err != nil {
		return err
	}
	averageGasPerSecond := (float64(latestBlockNumber) - g.epochStartBlockNumber) * g.averageBlockGasLimit
	g.gasPricer.CompleteEpoch(averageGasPerSecond)
	return g.updateL2GasPriceFn(g.gasPricer.curPrice)
}

func (g *GasPriceUpdater) GetGasPrice() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.gasPricer.curPrice
}
