package mon

import (
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

	t.Run("RecordsUnexpectedClaimResolution", func(t *testing.T) {
		monitor, cl, cMetrics := newTestClaimMonitor(t)
		games := makeMultipleTestGames(uint64(cl.Now().Unix()))
		monitor.CheckClaims(games)

		// Our honest actors 0x01 has claims resolved against them (1 per game)
		require.Equal(t, 2, cMetrics.invalid[common.Address{0x01}])
		require.Equal(t, 0, cMetrics.invalid[common.Address{0x02}])

		// Should report the number of valid claims
		require.Equal(t, 0, cMetrics.valid[common.Address{0x01}])
		require.Equal(t, 2, cMetrics.valid[common.Address{0x02}])

		// Should not have metrics for the actors not in the honest list
		require.NotContains(t, cMetrics.invalid, common.Address{0x03})
		require.NotContains(t, cMetrics.valid, common.Address{0x03})
		require.NotContains(t, cMetrics.invalid, common.Address{0x04})
		require.NotContains(t, cMetrics.valid, common.Address{0x04})
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
	calls   map[metrics.ClaimStatus]int
	invalid map[common.Address]int
	valid   map[common.Address]int
}

func (s *stubClaimMetrics) RecordClaims(status metrics.ClaimStatus, count int) {
	if s.calls == nil {
		s.calls = make(map[metrics.ClaimStatus]int)
	}
	s.calls[status] += count
}

func (s *stubClaimMetrics) RecordHonestActorClaimResolution(address common.Address, invalid int, valid int) {
	if s.invalid == nil {
		s.invalid = make(map[common.Address]int)
	}
	if s.valid == nil {
		s.valid = make(map[common.Address]int)
	}
	s.invalid[address] += invalid
	s.valid[address] += valid
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
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					Claimant:    common.Address{0x01},
					CounteredBy: common.Address{0x03},
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					Claimant:    common.Address{0x04},
					CounteredBy: common.Address{0x02},
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					Claimant:    common.Address{0x04},
					CounteredBy: common.Address{0x02},
					Clock:       faultTypes.NewClock(time.Duration(0), frozen),
				},
			},
			{
				Claim: faultTypes.Claim{
					Claimant: common.Address{0x01},
				},
			},
		},
	}
}
