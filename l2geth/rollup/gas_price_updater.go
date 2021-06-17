package rollup

type GetLatestBlockNumberFn func() (uint64, error)
type UpdateL2GasPriceFn func(float64) error

type GasPriceUpdater struct {
	gasPricer              L2GasPricer
	epochStartBlockNumber  uint64
	averageBlockGasLimit   uint64
	getLatestBlockNumberFn GetLatestBlockNumberFn
	updateL2GasPriceFn     UpdateL2GasPriceFn
}

func NewGasPriceUpdater(
	gasPricer *L2GasPricer,
	epochStartBlockNumber uint64,
	averageBlockGasLimit uint64,
	getLatestBlockNumberFn GetLatestBlockNumberFn,
	updateL2GasPriceFn UpdateL2GasPriceFn,
) *GasPriceUpdater {
	return &GasPriceUpdater{
		gasPricer:              *gasPricer,
		epochStartBlockNumber:  epochStartBlockNumber,
		averageBlockGasLimit:   averageBlockGasLimit,
		getLatestBlockNumberFn: getLatestBlockNumberFn,
		updateL2GasPriceFn:     updateL2GasPriceFn,
	}
}

func (g *GasPriceUpdater) UpdateGasPrice() error {
	latestBlockNumber, err := g.getLatestBlockNumberFn()
	if err != nil {
		return err
	}
	averageGasPerSecond := float64((latestBlockNumber - g.epochStartBlockNumber) * g.averageBlockGasLimit)
	g.gasPricer.CompleteEpoch(averageGasPerSecond)
	return g.updateL2GasPriceFn(g.gasPricer.curPrice)
}
