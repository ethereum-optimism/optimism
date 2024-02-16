package resolution

import (
	"math"
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

func clockFromParts(duration, timestamp uint64) *types.Clock {
	bigDuration := new(big.Int).SetUint64(duration)
	encoded := new(big.Int).Lsh(bigDuration, 64)
	raw := new(big.Int).Or(encoded, new(big.Int).SetUint64(timestamp))
	return types.NewClock(raw)
}

func TestDelayCalculator_getRemainingTime(t *testing.T) {
	t.Run("NoClock", func(t *testing.T) {
		d, metrics, _ := setupDelayCalculatorTest(t)
		claim := &types.Claim{
			ClaimData: types.ClaimData{
				Bond: monTypes.ResolvedBondAmount,
			},
		}
		delay := d.getRemainingTime(maxGameDuration, claim)
		require.Equal(t, uint64(0), delay)
		require.Equal(t, 0, metrics.calls)
	})

	t.Run("RemainingTime", func(t *testing.T) {
		d, metrics, cl := setupDelayCalculatorTest(t)
		duration := uint64(3 * 60)
		timestamp := uint64(cl.Now().Add(-time.Minute).Unix())
		claim := &types.Claim{
			ClaimData: types.ClaimData{
				Bond: big.NewInt(5),
			},
			Clock: clockFromParts(duration, timestamp),
		}
		delay := d.getRemainingTime(maxGameDuration, claim)
		require.Equal(t, uint64(240), delay)
		require.Equal(t, 0, metrics.calls)
	})

	t.Run("Overflows", func(t *testing.T) {
		d, metrics, cl := setupDelayCalculatorTest(t)
		duration := maxGameDuration / 2
		timestamp := uint64(cl.Now().Unix())
		claim := &types.Claim{
			ClaimData: types.ClaimData{
				Bond: big.NewInt(5),
			},
			Clock: clockFromParts(duration, timestamp),
		}
		delay := d.getRemainingTime(maxGameDuration, claim)
		require.Equal(t, uint64(0), delay)
		require.Equal(t, 0, metrics.calls)
	})
}

func TestDelayCalculator_getResolutionDelays(t *testing.T) {
	tests := []struct {
		name     string
		claims   []types.Claim
		minDelay uint64
		maxDelay uint64
	}{
		{"NoClaims", []types.Claim{}, math.MaxUint64, 0},
		{"SingleClaim", createClaimList()[:1], 180, 180},
		{"MultipleClaims", createClaimList()[:2], 180, 300},
		{"ClaimsWithMaxUint128", createClaimList(), 180, 300},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			d, metrics, _ := setupDelayCalculatorTest(t)
			game := &monTypes.EnrichedGameData{
				Claims:   test.claims,
				Duration: maxGameDuration,
			}
			minDelay, maxDelay := d.getResolutionDelays(game)
			require.Equal(t, 0, metrics.calls)
			require.Equal(t, test.minDelay, minDelay)
			require.Equal(t, test.maxDelay, maxDelay)
		})
	}
}

func TestDelayCalculator_RecordResolutionDelays(t *testing.T) {
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
			d.RecordResolutionDelays(test.games)
			require.Equal(t, 2, metrics.calls)
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

func createClaimList() []types.Claim {
	newClock := func(multiplier int) *types.Clock {
		duration := uint64(2 * 60)
		timestamp := uint64(frozen.Add(-time.Minute * time.Duration(multiplier)).Unix())
		return clockFromParts(duration, timestamp)
	}
	return []types.Claim{
		{
			ClaimData: types.ClaimData{
				Bond: big.NewInt(10),
			},
			Clock: newClock(3),
		},
		{
			ClaimData: types.ClaimData{
				Bond: big.NewInt(5),
			},
			Clock: newClock(1),
		},
		{
			ClaimData: types.ClaimData{
				Bond: big.NewInt(100),
			},
			Clock: newClock(2),
		},
		{
			// This claim should be skipped because it's resolved.
			ClaimData: types.ClaimData{
				Bond: monTypes.ResolvedBondAmount,
			},
			Clock: newClock(10),
		},
	}
}

type mockDelayMetrics struct {
	calls    int
	maxDelay float64
	minDelay float64
}

func (m *mockDelayMetrics) RecordClaimResolutionDelayMax(delay float64) {
	m.calls++
	if delay > m.maxDelay {
		m.maxDelay = delay
	}
}

func (m *mockDelayMetrics) RecordClaimResolutionDelayMin(delay float64) {
	m.calls++
	if delay < m.minDelay {
		m.minDelay = delay
	}
}
