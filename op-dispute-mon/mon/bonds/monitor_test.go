package bonds

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	frozen = time.Unix(int64(time.Hour.Seconds()), 0)
)

func TestCheckBonds(t *testing.T) {
	weth1 := common.Address{0x1a}
	weth1Balance := big.NewInt(4200)
	weth2 := common.Address{0x2b}
	weth2Balance := big.NewInt(6000)
	game1 := &monTypes.EnrichedGameData{
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(2),
		},
		WETHContract:  weth1,
		ETHCollateral: weth1Balance,
	}
	game2 := &monTypes.EnrichedGameData{
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(46),
		},
		WETHContract:  weth2,
		ETHCollateral: weth2Balance,
	}

	logger := testlog.Logger(t, log.LvlInfo)
	metrics := &stubBondMetrics{
		credits:  make(map[metrics.CreditExpectation]int),
		recorded: make(map[common.Address]Collateral),
	}
	bonds := NewBonds(logger, metrics, clock.NewDeterministicClock(frozen))

	bonds.CheckBonds([]*monTypes.EnrichedGameData{game1, game2})

	require.Len(t, metrics.recorded, 2)
	require.Contains(t, metrics.recorded, weth1)
	require.Contains(t, metrics.recorded, weth2)
	require.Equal(t, metrics.recorded[weth1].Required.Uint64(), uint64(2))
	require.Equal(t, metrics.recorded[weth1].Actual.Uint64(), weth1Balance.Uint64())
	require.Equal(t, metrics.recorded[weth2].Required.Uint64(), uint64(46))
	require.Equal(t, metrics.recorded[weth2].Actual.Uint64(), weth2Balance.Uint64())
}

type stubBondMetrics struct {
	credits  map[metrics.CreditExpectation]int
	recorded map[common.Address]Collateral
}

func (s *stubBondMetrics) RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int) {
	s.recorded[addr] = Collateral{
		Required: required,
		Actual:   available,
	}
}

func (s *stubBondMetrics) RecordCredit(expectation metrics.CreditExpectation, count int) {
	s.credits[expectation] = count
}
