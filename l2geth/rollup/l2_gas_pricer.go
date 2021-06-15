package rollup

import (
	"math"
)

type GetTargetGasPerSecond func() float64

type L2GasPricer struct {
	curPrice                 float64
	floorPrice               float64
	getTargetGasPerSecond    GetTargetGasPerSecond
	maxPercentChangePerEpoch float64
}

func NewGasPricer(curPrice float64, floorPrice float64, getTargetGasPerSecond GetTargetGasPerSecond, maxPercentChangePerEpoch float64) (L2GasPricer, error) {
	p := L2GasPricer{
		curPrice:                 math.Max(curPrice, floorPrice),
		floorPrice:               floorPrice,
		getTargetGasPerSecond:    getTargetGasPerSecond,
		maxPercentChangePerEpoch: maxPercentChangePerEpoch,
	}
	return p, nil
}

// Calculate the next gas price given some average gas per second over the last epoch
func (p *L2GasPricer) CalcNextGasPrice(avgGasPerSecondLastEpoch float64) float64 {
	avgGasPerSecondLastEpoch = math.Max(0, avgGasPerSecondLastEpoch)
	// The percent difference between our current average gas & our target gas
	proportionOfTarget := float64(avgGasPerSecondLastEpoch / p.getTargetGasPerSecond())
	// The percent that we should adjust the gas price to reach our target gas
	proportionToChangeBy := float64(0)
	if proportionOfTarget >= 1 { // If average avgGasPerSecondLastEpoch is GREATER than our target
		proportionToChangeBy = math.Min(proportionOfTarget, 1+p.maxPercentChangePerEpoch)
	} else {
		proportionToChangeBy = math.Max(proportionOfTarget, 1-p.maxPercentChangePerEpoch)
	}
	return math.Ceil(math.Max(p.floorPrice, p.curPrice*proportionToChangeBy))
}

// Update the gas price for this epoch
func (p *L2GasPricer) UpdateGasPrice(avgGasPerSecondLastEpoch float64) {
	p.curPrice = p.CalcNextGasPrice(avgGasPerSecondLastEpoch)
}
