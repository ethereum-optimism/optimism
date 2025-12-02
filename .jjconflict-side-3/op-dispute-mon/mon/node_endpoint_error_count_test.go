package mon

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestCheckNodeEndpointErrorCount_NoErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, RollupEndpointErrorCount: 0},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, RollupEndpointErrorCount: 0},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}}, RollupEndpointErrorCount: 0},
	}

	metrics := &stubNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrorCount(games)

	require.Equal(t, 0, metrics.recordedCount)
}

func TestCheckNodeEndpointErrorCount_SingleGameWithErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointErrorCount: 5,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointErrorCount: 0,
		},
	}

	metrics := &stubNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrorCount(games)

	require.Equal(t, 5, metrics.recordedCount)
}

func TestCheckNodeEndpointErrorCount_MultipleGamesWithErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointErrorCount: 3,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointErrorCount: 7,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointErrorCount: 2,
		},
	}

	metrics := &stubNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrorCount(games)

	// Should sum all error counts (3 + 7 + 2 = 12)
	require.Equal(t, 12, metrics.recordedCount)
}

func TestCheckNodeEndpointErrorCount_MixedGamesWithAndWithoutErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointErrorCount: 0,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointErrorCount: 4,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointErrorCount: 0,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x44}},
			RollupEndpointErrorCount: 6,
		},
	}

	metrics := &stubNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrorCount(games)

	// Should sum only non-zero error counts (4 + 6 = 10)
	require.Equal(t, 10, metrics.recordedCount)
}

func TestCheckNodeEndpointErrorCount_EmptyGamesList(t *testing.T) {
	games := []*types.EnrichedGameData{}

	metrics := &stubNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrorCount(games)

	require.Equal(t, 0, metrics.recordedCount)
}

func TestCheckNodeEndpointErrorCount_HighVolumeErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointErrorCount: 100,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointErrorCount: 250,
		},
		{
			GameMetadata:             gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointErrorCount: 75,
		},
	}

	metrics := &stubNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrorCount(games)

	// Should sum all error counts (100 + 250 + 75 = 425)
	require.Equal(t, 425, metrics.recordedCount)
}

func TestCountGamesWithErrors(t *testing.T) {
	tests := []struct {
		name     string
		games    []*types.EnrichedGameData
		expected int
	}{
		{
			name:     "no games",
			games:    []*types.EnrichedGameData{},
			expected: 0,
		},
		{
			name: "no errors",
			games: []*types.EnrichedGameData{
				{RollupEndpointErrorCount: 0},
				{RollupEndpointErrorCount: 0},
			},
			expected: 0,
		},
		{
			name: "all games have errors",
			games: []*types.EnrichedGameData{
				{RollupEndpointErrorCount: 1},
				{RollupEndpointErrorCount: 5},
				{RollupEndpointErrorCount: 10},
			},
			expected: 3,
		},
		{
			name: "mixed errors",
			games: []*types.EnrichedGameData{
				{RollupEndpointErrorCount: 0},
				{RollupEndpointErrorCount: 3},
				{RollupEndpointErrorCount: 0},
				{RollupEndpointErrorCount: 7},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countGamesWithErrors(tt.games)
			require.Equal(t, tt.expected, result)
		})
	}
}

type stubNodeEndpointErrorCountMetrics struct {
	recordedCount int
}

func (s *stubNodeEndpointErrorCountMetrics) RecordNodeEndpointErrorCount(count int) {
	s.recordedCount = count
}
