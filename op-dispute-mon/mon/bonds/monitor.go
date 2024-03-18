package bonds

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/transform"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type BondMetrics interface {
	RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int)
}

type Bonds struct {
	logger  log.Logger
	metrics BondMetrics
}

func NewBonds(logger log.Logger, metrics BondMetrics) *Bonds {
	return &Bonds{
		logger:  logger,
		metrics: metrics,
	}
}

func (b *Bonds) CheckBonds(games []*types.EnrichedGameData) {
	data := transform.CalculateRequiredCollateral(games)
	for addr, collateral := range data {
		b.metrics.RecordBondCollateral(addr, collateral.Required, collateral.Actual)
	}
}
