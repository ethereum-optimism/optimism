package mon

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
		monitor, cl, cMetrics, _ := newTestClaimMonitor(t)
		games := makeMultipleTestGames(uint64(cl.Now().Unix()))
		monitor.CheckClaims(games)

		for status, count := range cMetrics.calls {
			fmt.Printf("%v: %v \n", status, count)
		}

		// Test data is a bit weird and has unresolvable claims that have been resolved
		require.Equal(t, 2, cMetrics.calls[metrics.NewClaimStatus(true, true, false, true)])
		require.Equal(t, 1, cMetrics.calls[metrics.NewClaimStatus(true, true, true, false)])
		require.Equal(t, 1, cMetrics.calls[metrics.NewClaimStatus(true, false, false, true)])
		require.Equal(t, 1, cMetrics.calls[metrics.NewClaimStatus(true, false, false, false)])

		// Test data is a bit weird and has unresolvable claims that have been resolved
		require.Equal(t, 2, cMetrics.calls[metrics.NewClaimStatus(false, true, false, true)])
		require.Equal(t, 1, cMetrics.calls[metrics.NewClaimStatus(false, true, true, false)])
		require.Equal(t, 1, cMetrics.calls[metrics.NewClaimStatus(false, false, false, true)])
		require.Equal(t, 1, cMetrics.calls[metrics.NewClaimStatus(false, false, false, false)])
	})

	t.Run("ZeroRecordsClaims", func(t *testing.T) {
		monitor, _, cMetrics, _ := newTestClaimMonitor(t)
		var games []*types.EnrichedGameData
		monitor.CheckClaims(games)
		// Should record 0 values for true and false variants of the four fields in ClaimStatus
		require.Len(t, cMetrics.calls, 2*2*2*2)
	})

	t.Run("ConsiderChildResolvability", func(t *testing.T) {
		monitor, _, cMetrics, logs := newTestClaimMonitor(t)
		chessClockDuration := 10 * time.Minute
		// Game started long enough ago that the root chess clock has now expired
		gameStart := frozen.Add(-chessClockDuration - 15*time.Minute)
		games := []*types.EnrichedGameData{
			{
				MaxClockDuration: uint64(chessClockDuration.Seconds()),
				GameMetadata: gameTypes.GameMetadata{
					Proxy:     common.Address{0xaa},
					Timestamp: 50,
				},
				Claims: []types.EnrichedClaim{
					{
						Claim: faultTypes.Claim{
							ContractIndex: 0,
							ClaimData: faultTypes.ClaimData{
								Position: faultTypes.RootPosition,
							},
							Clock: faultTypes.NewClock(time.Duration(0), gameStart),
						},
						Resolved: false,
					},
					{
						Claim: faultTypes.Claim{ // Fast challenge, clock has expired
							ContractIndex:       1,
							ParentContractIndex: 0,
							ClaimData: faultTypes.ClaimData{
								Position: faultTypes.RootPosition,
							},
							Clock: faultTypes.NewClock(1*time.Minute, gameStart.Add(1*time.Minute)),
						},
						Resolved: false,
					},
					{
						Claim: faultTypes.Claim{ // Fast counter to fast challenge, clock has expired, resolved
							ContractIndex:       2,
							ParentContractIndex: 1,
							ClaimData: faultTypes.ClaimData{
								Position: faultTypes.RootPosition,
							},
							Clock: faultTypes.NewClock(1*time.Minute, gameStart.Add((1+1)*time.Minute)),
						},
						Resolved: true,
					},
					{
						Claim: faultTypes.Claim{ // Second fast counter to fast challenge, clock has expired, not resolved
							ContractIndex:       3,
							ParentContractIndex: 1,
							ClaimData: faultTypes.ClaimData{
								Position: faultTypes.RootPosition,
							},
							Clock: faultTypes.NewClock(1*time.Minute, gameStart.Add((1+1)*time.Minute)),
						},
						Resolved: false,
					},
					{
						Claim: faultTypes.Claim{ // Challenge, clock has not yet expired
							ContractIndex:       4,
							ParentContractIndex: 0,
							ClaimData: faultTypes.ClaimData{
								Position: faultTypes.RootPosition,
							},
							Clock: faultTypes.NewClock(20*time.Minute, gameStart.Add(20*time.Minute)),
						},
						Resolved: false,
					},
					{
						Claim: faultTypes.Claim{ // Counter to challenge, clock hasn't expired yet
							ContractIndex:       5,
							ParentContractIndex: 4,
							ClaimData: faultTypes.ClaimData{
								Position: faultTypes.RootPosition,
							},
							Clock: faultTypes.NewClock(1*time.Minute, gameStart.Add((20+1)*time.Minute)),
						},
						Resolved: false,
					},
				},
			},
		}
		monitor.CheckClaims(games)
		expected := &metrics.ClaimStatuses{}
		// Root claim - clock expired, but not resolvable because of child claims
		expected.RecordClaim(false, true, false, false)
		// Claim 1 - clock expired, resolvable as both children are resolvable even though only one is resolved
		expected.RecordClaim(false, true, true, false)
		// Claim 2 - clock expired, resolvable and resolved
		expected.RecordClaim(false, true, true, true)
		// Claim 3 - clock expired, resolvable but not resolved
		expected.RecordClaim(false, true, true, false)
		// Claim 4 - clock not expired
		expected.RecordClaim(false, false, false, false)
		// Claim 5 - clock not expired
		expected.RecordClaim(false, false, false, false)

		expected.ForEachStatus(func(status metrics.ClaimStatus, count int) {
			require.Equalf(t, count, cMetrics.calls[status], "status %v", status)
		})

		unresolvedClaimMsg := testlog.NewMessageFilter("Claim unresolved after clock expiration")
		claim1Warn := logs.FindLog(unresolvedClaimMsg, testlog.NewAttributesFilter("claimContractIndex", "1"))
		require.NotNil(t, claim1Warn, "Should warn about claim 1 being unresolved")
		claim3Warn := logs.FindLog(unresolvedClaimMsg, testlog.NewAttributesFilter("claimContractIndex", "3"))
		require.NotNil(t, claim3Warn, "Should warn about claim 3 being unresolved")

		require.Equal(t, claim3Warn.AttrValue("delay"), claim1Warn.AttrValue("delay"),
			"Claim 1 should have same delay as claim 3 as it could not be resolved before claim 3 clock expired")
	})

	t.Run("RecordsUnexpectedClaimResolution", func(t *testing.T) {
		monitor, cl, cMetrics, _ := newTestClaimMonitor(t)
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

func newTestClaimMonitor(t *testing.T) (*ClaimMonitor, *clock.DeterministicClock, *stubClaimMetrics, *testlog.CapturingHandler) {
	logger, handler := testlog.CaptureLogger(t, log.LvlInfo)
	cl := clock.NewDeterministicClock(frozen)
	metrics := &stubClaimMetrics{}
	honestActors := types.NewHonestActors([]common.Address{
		{0x01},
		{0x02},
	})
	monitor := NewClaimMonitor(logger, cl, honestActors, metrics)
	return monitor, cl, metrics, handler
}

type stubClaimMetrics struct {
	calls  map[metrics.ClaimStatus]int
	honest map[common.Address]metrics.HonestActorData
}

func (s *stubClaimMetrics) RecordClaims(statuses *metrics.ClaimStatuses) {
	if s.calls == nil {
		s.calls = make(map[metrics.ClaimStatus]int)
	}
	statuses.ForEachStatus(func(status metrics.ClaimStatus, count int) {
		s.calls[status] = count
	})
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
