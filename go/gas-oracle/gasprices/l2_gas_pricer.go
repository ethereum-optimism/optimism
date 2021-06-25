package gasprices

import (
	"math"

	"github.com/ethereum/go-ethereum/log"
)

type GetTargetGasPerSecond func() float64

// This is not thread safe
type L2GasPricer struct {
	curPrice                 float64
	floorPrice               float64
	getTargetGasPerSecond    GetTargetGasPerSecond
	maxPercentChangePerEpoch float64
}

// LinearInterpolation can be used to dynamically update target gas per second
func GetLinearInterpolationFn(getX func() float64, x1 float64, x2 float64, y1 float64, y2 float64) func() float64 {
	return func() float64 {
		return y1 + ((getX()-x1)/(x2-x1))*(y2-y1)
	}
}

func NewGasPricer(curPrice float64, floorPrice float64, getTargetGasPerSecond GetTargetGasPerSecond, maxPercentChangePerEpoch float64) *L2GasPricer {
	return &L2GasPricer{
		curPrice:                 math.Max(curPrice, floorPrice),
		floorPrice:               floorPrice,
		getTargetGasPerSecond:    getTargetGasPerSecond,
		maxPercentChangePerEpoch: maxPercentChangePerEpoch,
	}
}

// Calculate the next gas price given some average gas per second over the last epoch
func (p *L2GasPricer) CalcNextEpochGasPrice(avgGasPerSecondLastEpoch float64) float64 {
	avgGasPerSecondLastEpoch = math.Max(0, avgGasPerSecondLastEpoch)
	// The percent difference between our current average gas & our target gas
	proportionOfTarget := avgGasPerSecondLastEpoch / p.getTargetGasPerSecond()
	// The percent that we should adjust the gas price to reach our target gas
	proportionToChangeBy := float64(0)
	if proportionOfTarget >= 1 { // If average avgGasPerSecondLastEpoch is GREATER than our target
		proportionToChangeBy = math.Min(proportionOfTarget, 1+p.maxPercentChangePerEpoch)
	} else {
		proportionToChangeBy = math.Max(proportionOfTarget, 1-p.maxPercentChangePerEpoch)
	}
	log.Debug("CalcNextEpochGasPrice", "proportionToChangeBy", proportionToChangeBy, "proportionOfTarget", proportionOfTarget)
	return math.Ceil(math.Max(p.floorPrice, math.Max(1, p.curPrice)*proportionToChangeBy))
}

// End the current epoch and update the current gas price for the next epoch.
func (p *L2GasPricer) CompleteEpoch(avgGasPerSecondLastEpoch float64) uint64 {
	p.curPrice = p.CalcNextEpochGasPrice(avgGasPerSecondLastEpoch)
	return uint64(p.curPrice)
}
