package gasprices

import (
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/log"
)

type GetTargetGasPerSecond func() float64

type GasPricer struct {
	curPrice                 uint64
	avgGasPerSecondLastEpoch float64
	floorPrice               uint64
	getTargetGasPerSecond    GetTargetGasPerSecond
	maxChangePerEpoch        float64
}

// LinearInterpolation can be used to dynamically update target gas per second
func GetLinearInterpolationFn(getX func() float64, x1 float64, x2 float64, y1 float64, y2 float64) func() float64 {
	return func() float64 {
		return y1 + ((getX()-x1)/(x2-x1))*(y2-y1)
	}
}

// NewGasPricer creates a GasPricer and checks its config beforehand
func NewGasPricer(curPrice, floorPrice uint64, getTargetGasPerSecond GetTargetGasPerSecond, maxPercentChangePerEpoch float64) (*GasPricer, error) {
	if floorPrice < 1 {
		return nil, errors.New("floorPrice must be greater than or equal to 1")
	}
	if maxPercentChangePerEpoch <= 0 {
		return nil, errors.New("maxPercentChangePerEpoch must be between (0,100]")
	}
	return &GasPricer{
		curPrice:              max(curPrice, floorPrice),
		floorPrice:            floorPrice,
		getTargetGasPerSecond: getTargetGasPerSecond,
		maxChangePerEpoch:     maxPercentChangePerEpoch,
	}, nil
}

// CalcNextEpochGasPrice calculates the next gas price given some average
// gas per second over the last epoch
func (p *GasPricer) CalcNextEpochGasPrice(avgGasPerSecondLastEpoch float64) (uint64, error) {
	targetGasPerSecond := p.getTargetGasPerSecond()
	if avgGasPerSecondLastEpoch < 0 {
		return 0.0, fmt.Errorf("avgGasPerSecondLastEpoch cannot be negative, got %f", avgGasPerSecondLastEpoch)
	}
	if targetGasPerSecond < 1 {
		return 0.0, fmt.Errorf("gasPerSecond cannot be less than 1, got %f", targetGasPerSecond)
	}
	// The percent difference between our current average gas & our target gas
	proportionOfTarget := avgGasPerSecondLastEpoch / targetGasPerSecond

	log.Trace("Calculating next epoch gas price", "proportionOfTarget", proportionOfTarget,
		"avgGasPerSecondLastEpoch", avgGasPerSecondLastEpoch, "targetGasPerSecond", targetGasPerSecond)

	// The percent that we should adjust the gas price to reach our target gas
	proportionToChangeBy := 0.0
	if proportionOfTarget >= 1 { // If average avgGasPerSecondLastEpoch is GREATER than our target
		proportionToChangeBy = math.Min(proportionOfTarget, 1+p.maxChangePerEpoch)
	} else {
		proportionToChangeBy = math.Max(proportionOfTarget, 1-p.maxChangePerEpoch)
	}

	updated := float64(max(1, p.curPrice)) * proportionToChangeBy
	result := max(p.floorPrice, uint64(math.Ceil(updated)))

	log.Debug("Calculated next epoch gas price", "proportionToChangeBy", proportionToChangeBy,
		"proportionOfTarget", proportionOfTarget, "result", result)

	return result, nil
}

// CompleteEpoch ends the current epoch and updates the current gas price for the next epoch
func (p *GasPricer) CompleteEpoch(avgGasPerSecondLastEpoch float64) (uint64, error) {
	gp, err := p.CalcNextEpochGasPrice(avgGasPerSecondLastEpoch)
	if err != nil {
		return gp, err
	}
	p.curPrice = gp
	p.avgGasPerSecondLastEpoch = avgGasPerSecondLastEpoch
	return gp, nil
}

func max(a, b uint64) uint64 {
	if a >= b {
		return a
	}
	return b
}
