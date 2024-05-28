package mon

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	weth1 = common.Address{0x1a}
	weth2 = common.Address{0x2b}
)

func makeGames() []*monTypes.EnrichedGameData {
	weth1Balance := big.NewInt(4200)
	weth2Balance := big.NewInt(6000)
	game1 := &monTypes.EnrichedGameData{
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(3),
			common.Address{0x02}: big.NewInt(1),
		},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			common.Address{0x01}: &contracts.WithdrawalRequest{Amount: big.NewInt(3)},
			common.Address{0x02}: &contracts.WithdrawalRequest{Amount: big.NewInt(1)},
		},
		WETHContract:  weth1,
		ETHCollateral: weth1Balance,
	}
	game2 := &monTypes.EnrichedGameData{
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(46),
			common.Address{0x02}: big.NewInt(1),
		},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			common.Address{0x01}: &contracts.WithdrawalRequest{Amount: big.NewInt(3)},
			common.Address{0x02}: &contracts.WithdrawalRequest{Amount: big.NewInt(1)},
		},
		WETHContract:  weth2,
		ETHCollateral: weth2Balance,
	}
	game3 := &monTypes.EnrichedGameData{
		Credits: map[common.Address]*big.Int{
			common.Address{0x03}: big.NewInt(2),
			common.Address{0x04}: big.NewInt(4),
		},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			common.Address{0x03}: &contracts.WithdrawalRequest{Amount: big.NewInt(2)},
			common.Address{0x04}: &contracts.WithdrawalRequest{Amount: big.NewInt(4)},
		},
		WETHContract:  weth2,
		ETHCollateral: weth2Balance,
	}
	return []*monTypes.EnrichedGameData{game1, game2, game3}
}

func TestCheckWithdrawals(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	metrics := &stubWithdrawalsMetrics{
		matching:  make(map[common.Address]int),
		divergent: make(map[common.Address]int),
	}
	withdrawals := NewWithdrawalMonitor(logger, metrics)
	withdrawals.CheckWithdrawals(makeGames())

	require.Equal(t, metrics.matchCalls, 2)
	require.Equal(t, metrics.divergeCalls, 2)
	require.Len(t, metrics.matching, 2)
	require.Len(t, metrics.divergent, 2)
	require.Contains(t, metrics.matching, weth1)
	require.Contains(t, metrics.matching, weth2)
	require.Contains(t, metrics.divergent, weth1)
	require.Contains(t, metrics.divergent, weth2)
	require.Equal(t, metrics.matching[weth1], 2)
	require.Equal(t, metrics.matching[weth2], 3)
	require.Equal(t, metrics.divergent[weth1], 0)
	require.Equal(t, metrics.divergent[weth2], 1)
}

type stubWithdrawalsMetrics struct {
	matchCalls   int
	divergeCalls int
	matching     map[common.Address]int
	divergent    map[common.Address]int
}

func (s *stubWithdrawalsMetrics) RecordWithdrawalRequests(addr common.Address, matches bool, count int) {
	if matches {
		s.matchCalls++
		s.matching[addr] = count
	} else {
		s.divergeCalls++
		s.divergent[addr] = count
	}
}
