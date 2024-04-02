package mon

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var frozen = time.Unix(int64(time.Hour.Seconds()), 0)

func TestClaimMonitor_CheckClaims(t *testing.T) {
	cm, cl, cMetrics := newTestClaimMonitor(t)
	games := makeMultipleTestGames(uint64(cl.Now().Unix()))
	cm.CheckClaims(games)

	require.Equal(t, 1, cMetrics.calls[metrics.FirstHalfExpiredResolved])
	require.Equal(t, 1, cMetrics.calls[metrics.FirstHalfExpiredUnresolved])
	require.Equal(t, 1, cMetrics.calls[metrics.FirstHalfNotExpiredResolved])
	require.Equal(t, 1, cMetrics.calls[metrics.FirstHalfNotExpiredUnresolved])

	require.Equal(t, 1, cMetrics.calls[metrics.SecondHalfExpiredResolved])
	require.Equal(t, 1, cMetrics.calls[metrics.SecondHalfExpiredUnresolved])
	require.Equal(t, 1, cMetrics.calls[metrics.SecondHalfNotExpiredResolved])
	require.Equal(t, 1, cMetrics.calls[metrics.SecondHalfNotExpiredUnresolved])
}

func newTestClaimMonitor(t *testing.T) (*ClaimMonitor, *clock.DeterministicClock, *stubClaimMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	cl := clock.NewDeterministicClock(frozen)
	metrics := &stubClaimMetrics{}
	return NewClaimMonitor(logger, cl, metrics), cl, metrics
}

type stubClaimMetrics struct {
	calls map[metrics.ClaimStatus]int
}

func (s *stubClaimMetrics) RecordClaims(status metrics.ClaimStatus, count int) {
	if s.calls == nil {
		s.calls = make(map[metrics.ClaimStatus]int)
	}
	s.calls[status] += count
}

func makeMultipleTestGames(duration uint64) []*types.EnrichedGameData {
	return []*types.EnrichedGameData{
		makeTestGame(duration), // first half
		makeTestGame(duration * 10), // second half
	}
}

func makeTestGame(duration uint64) *types.EnrichedGameData {
	return &types.EnrichedGameData{
		Duration: duration,
		Recipients: map[common.Address]bool{
			common.Address{0x02}: true,
			common.Address{0x03}: true,
			common.Address{0x04}: true,
		},
		Claims: []types.EnrichedClaim{
			{
				Claim: faultTypes.Claim{
					Clock: faultTypes.NewClock(time.Duration(0), frozen),
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					Clock: faultTypes.NewClock(time.Duration(0), frozen),
				},
			},
			{
				Claim: faultTypes.Claim{},
			},
		},
	}
}
