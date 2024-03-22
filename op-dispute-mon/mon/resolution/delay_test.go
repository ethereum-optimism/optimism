package resolution

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/stretchr/testify/require"
)

var (
	maxGameDuration = uint64(960)
	frozen          = time.Unix(int64(time.Hour.Seconds()), 0)
)

func TestDelayCalculator_getOverflowTime(t *testing.T) {
	t.Run("NoClock", func(t *testing.T) {
		d, metrics, _ := setupDelayCalculatorTest(t)
		claim := &monTypes.EnrichedClaim{
			Resolved: true,
		}
		delay := d.getOverflowTime(maxGameDuration, claim)
		require.Equal(t, uint64(0), delay)
		require.Equal(t, 0, metrics.calls)
	})

	t.Run("RemainingTime", func(t *testing.T) {
		d, metrics, cl := setupDelayCalculatorTest(t)
		duration := 3 * time.Minute
		timestamp := cl.Now().Add(-time.Minute)
		claim := &monTypes.EnrichedClaim{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Bond: big.NewInt(5),
				},
				Clock: types.NewClock(duration, timestamp),
			},
		}
		delay := d.getOverflowTime(maxGameDuration, claim)
		require.Equal(t, uint64(0), delay)
		require.Equal(t, 0, metrics.calls)
	})

	t.Run("OverflowTime", func(t *testing.T) {
		d, metrics, cl := setupDelayCalculatorTest(t)
		duration := time.Duration(maxGameDuration/2) * time.Second
		timestamp := cl.Now().Add(4 * -time.Minute)
		claim := &monTypes.EnrichedClaim{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Bond: big.NewInt(5),
				},
				Clock: types.NewClock(duration, timestamp),
			},
		}
		delay := d.getOverflowTime(maxGameDuration, claim)
		require.Equal(t, uint64(240), delay)
		require.Equal(t, 0, metrics.calls)
	})
}

func TestDelayCalculator_getMaxResolutionDelay(t *testing.T) {
	tests := []struct {
		name   string
		claims []monTypes.EnrichedClaim
		want   uint64
	}{
		{"NoClaims", []monTypes.EnrichedClaim{}, 0},
		{"SingleClaim", createClaimList()[:1], 180},
		{"MultipleClaims", createClaimList()[:2], 300},
		{"ClaimsWithMaxUint128", createClaimList(), 300},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			d, metrics, _ := setupDelayCalculatorTest(t)
			game := &monTypes.EnrichedGameData{
				Claims:   test.claims,
				Duration: maxGameDuration,
			}
			got := d.getMaxResolutionDelay(game)
			require.Equal(t, 0, metrics.calls)
			require.Equal(t, test.want, got)
		})
	}
}

func TestDelayCalculator_RecordClaimResolutionDelayMax(t *testing.T) {
	tests := []struct {
		name  string
		games []*monTypes.EnrichedGameData
		want  float64
	}{
		{"NoGames", createGameWithClaimsList()[:0], 0},
		{"SingleGame", createGameWithClaimsList()[:1], 180},
		{"MultipleGames", createGameWithClaimsList()[:2], 300},
		{"ClaimsWithMaxUint128", createGameWithClaimsList(), 300},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			d, metrics, _ := setupDelayCalculatorTest(t)
			d.RecordClaimResolutionDelayMax(test.games)
			require.Equal(t, 1, metrics.calls)
			require.Equal(t, test.want, metrics.maxDelay)
		})
	}
}

func setupDelayCalculatorTest(t *testing.T) (*DelayCalculator, *mockDelayMetrics, *clock.DeterministicClock) {
	metrics := &mockDelayMetrics{}
	cl := clock.NewDeterministicClock(frozen)
	return NewDelayCalculator(metrics, cl), metrics, cl
}

func createGameWithClaimsList() []*monTypes.EnrichedGameData {
	return []*monTypes.EnrichedGameData{
		{
			Claims:   createClaimList()[:1],
			Duration: maxGameDuration,
		},
		{
			Claims:   createClaimList()[:2],
			Duration: maxGameDuration,
		},
		{
			Claims:   createClaimList(),
			Duration: maxGameDuration,
		},
	}
}

func createClaimList() []monTypes.EnrichedClaim {
	newClock := func(multiplier int) types.Clock {
		duration := maxGameDuration / 2
		timestamp := frozen.Add(-time.Minute * time.Duration(multiplier))
		return types.NewClock(time.Duration(duration)*time.Second, timestamp)
	}
	return []monTypes.EnrichedClaim{
		{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Bond: big.NewInt(5),
				},
				Clock: newClock(3),
			},
		},
		{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Bond: big.NewInt(10),
				},
				Clock: newClock(5),
			},
		},
		{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Bond: big.NewInt(100),
				},
				Clock: newClock(2),
			},
		},
		{
			Claim: types.Claim{
				Clock: newClock(10),
			},
			Resolved: true,
		},
	}
}

type mockDelayMetrics struct {
	calls    int
	maxDelay float64
}

func (m *mockDelayMetrics) RecordClaimResolutionDelayMax(delay float64) {
	m.calls++
	if delay > m.maxDelay {
		m.maxDelay = delay
	}
}
