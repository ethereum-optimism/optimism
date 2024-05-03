package mon

import (
	"math/big"
	"testing"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var frozen = time.Unix(int64(time.Hour.Seconds()), 0)

func TestClaimMonitor_CheckClaims(t *testing.T) {
	t.Run("RecordsClaims", func(t *testing.T) {
		monitor, cl, cMetrics := newTestClaimMonitor(t)
		games := makeMultipleTestGames(uint64(cl.Now().Unix()))
		monitor.CheckClaims(games)

		require.Equal(t, 2, cMetrics.calls[metrics.FirstHalfExpiredResolved])
		require.Equal(t, 1, cMetrics.calls[metrics.FirstHalfExpiredUnresolved])
		require.Equal(t, 1, cMetrics.calls[metrics.FirstHalfNotExpiredResolved])
		require.Equal(t, 1, cMetrics.calls[metrics.FirstHalfNotExpiredUnresolved])

		require.Equal(t, 2, cMetrics.calls[metrics.SecondHalfExpiredResolved])
		require.Equal(t, 1, cMetrics.calls[metrics.SecondHalfExpiredUnresolved])
		require.Equal(t, 1, cMetrics.calls[metrics.SecondHalfNotExpiredResolved])
		require.Equal(t, 1, cMetrics.calls[metrics.SecondHalfNotExpiredUnresolved])
	})

	t.Run("ZeroRecordsClaims", func(t *testing.T) {
		monitor, _, cMetrics := newTestClaimMonitor(t)
		var games []*types.EnrichedGameData
		monitor.CheckClaims(games)
		// Check we zero'd out any categories that didn't have games in them (otherwise they retain their previous value)
		require.Contains(t, cMetrics.calls, metrics.FirstHalfExpiredResolved)
		require.Contains(t, cMetrics.calls, metrics.FirstHalfExpiredUnresolved)
		require.Contains(t, cMetrics.calls, metrics.FirstHalfNotExpiredResolved)
		require.Contains(t, cMetrics.calls, metrics.FirstHalfNotExpiredUnresolved)

		require.Contains(t, cMetrics.calls, metrics.SecondHalfExpiredResolved)
		require.Contains(t, cMetrics.calls, metrics.SecondHalfExpiredUnresolved)
		require.Contains(t, cMetrics.calls, metrics.SecondHalfNotExpiredResolved)
		require.Contains(t, cMetrics.calls, metrics.SecondHalfNotExpiredUnresolved)
	})

	t.Run("RecordsUnexpectedClaimResolution", func(t *testing.T) {
		monitor, cl, cMetrics := newTestClaimMonitor(t)
		games := makeMultipleTestGames(uint64(cl.Now().Unix()))
		monitor.CheckClaims(games)

		// Should only have entries for honest actors
		require.Contains(t, cMetrics.honest, common.Address{0x01})
		require.Contains(t, cMetrics.honest, common.Address{0x02})
		require.NotContains(t, cMetrics.honest, common.Address{0x03})
		require.NotContains(t, cMetrics.honest, common.Address{0x04})

		actor1 := cMetrics.honest[common.Address{0x01}]
		actor2 := cMetrics.honest[common.Address{0x02}]
		// Our honest actors 0x01 has claims resolved against them (1 per game)
		require.Equal(t, 2, actor1.InvalidClaimCount)
		require.Equal(t, 0, actor1.ValidClaimCount)
		require.Equal(t, 2, actor1.PendingClaimCount)
		require.EqualValues(t, 4, actor1.LostBonds.Int64())
		require.EqualValues(t, 0, actor1.WonBonds.Int64())
		require.EqualValues(t, 10, actor1.PendingBonds.Int64())

		require.Equal(t, 0, actor2.InvalidClaimCount)
		require.Equal(t, 2, actor2.ValidClaimCount)
		require.Equal(t, 0, actor2.PendingClaimCount)
		require.EqualValues(t, 0, actor2.LostBonds.Int64())
		require.EqualValues(t, 6, actor2.WonBonds.Int64())
		require.EqualValues(t, 0, actor2.PendingBonds.Int64())
	})
}

func newTestClaimMonitor(t *testing.T) (*ClaimMonitor, *clock.DeterministicClock, *stubClaimMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	cl := clock.NewDeterministicClock(frozen)
	metrics := &stubClaimMetrics{}
	honestActors := []common.Address{
		{0x01},
		{0x02},
	}
	return NewClaimMonitor(logger, cl, honestActors, metrics), cl, metrics
}

type stubClaimMetrics struct {
	calls  map[metrics.ClaimStatus]int
	honest map[common.Address]metrics.HonestActorData
}

func (s *stubClaimMetrics) RecordClaims(status metrics.ClaimStatus, count int) {
	if s.calls == nil {
		s.calls = make(map[metrics.ClaimStatus]int)
	}
	s.calls[status] += count
}

func (s *stubClaimMetrics) RecordHonestActorClaims(address common.Address, data *metrics.HonestActorData) {
	if s.honest == nil {
		s.honest = make(map[common.Address]metrics.HonestActorData)
	}
	s.honest[address] = *data
}

func makeMultipleTestGames(duration uint64) []*types.EnrichedGameData {
	return []*types.EnrichedGameData{
		makeTestGame(duration),      // first half
		makeTestGame(duration * 10), // second half
	}
}

func makeTestGame(duration uint64) *types.EnrichedGameData {
	return &types.EnrichedGameData{
		MaxClockDuration: duration / 2,
		Recipients: map[common.Address]bool{
			{0x02}: true,
			{0x03}: true,
			{0x04}: true,
		},
		Claims: []types.EnrichedClaim{
			{
				Claim: faultTypes.Claim{
					Clock:    faultTypes.NewClock(time.Duration(0), frozen),
					Claimant: common.Address{0x02},
					ClaimData: faultTypes.ClaimData{
						Bond: big.NewInt(1),
					},
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					Claimant:    common.Address{0x01},
					CounteredBy: common.Address{0x03},
					ClaimData: faultTypes.ClaimData{
						Bond: big.NewInt(2),
					},
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					Claimant:    common.Address{0x04},
					CounteredBy: common.Address{0x02},
					ClaimData: faultTypes.ClaimData{
						Bond: big.NewInt(3),
					},
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					Claimant:    common.Address{0x04},
					CounteredBy: common.Address{0x02},
					Clock:       faultTypes.NewClock(time.Duration(0), frozen),
					ClaimData: faultTypes.ClaimData{
						Bond: big.NewInt(4),
					},
				},
			},
			{
				Claim: faultTypes.Claim{
					Claimant: common.Address{0x01},
					ClaimData: faultTypes.ClaimData{
						Bond: big.NewInt(5),
					},
				},
			},
		},
	}
}
