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

func TestCheckNodeEndpointErrors_NoErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, RollupEndpointErrors: nil},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, RollupEndpointErrors: make(map[string]bool)},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}}}, // No RollupEndpointErrors field set
	}

	metrics := &stubNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	require.Equal(t, 0, metrics.recordedCount)
}

func TestCheckNodeEndpointErrors_SingleGameWithErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointErrors: map[string]bool{
				"endpoint_1": true,
				"endpoint_2": true,
			},
		},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, RollupEndpointErrors: nil},
	}

	metrics := &stubNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	require.Equal(t, 2, metrics.recordedCount)
}

func TestCheckNodeEndpointErrors_MultipleGamesWithOverlappingErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointErrors: map[string]bool{
				"endpoint_1": true,
				"endpoint_2": true,
			},
		},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointErrors: map[string]bool{
				"endpoint_2": true, // Overlapping with first game
				"endpoint_3": true,
			},
		},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointErrors: map[string]bool{
				"endpoint_4": true,
			},
		},
	}

	metrics := &stubNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	// Should count unique endpoints across all games (endpoint_1, endpoint_2, endpoint_3, endpoint_4)
	require.Equal(t, 4, metrics.recordedCount)
}

func TestCheckNodeEndpointErrors_MixedGamesWithAndWithoutErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, RollupEndpointErrors: nil},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointErrors: map[string]bool{
				"endpoint_1": true,
			},
		},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}}, RollupEndpointErrors: make(map[string]bool)},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x44}},
			RollupEndpointErrors: map[string]bool{
				"endpoint_2": true,
			},
		},
	}

	metrics := &stubNodeEndpointErrorsMetrics{}
	logger, _ := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	require.Equal(t, 2, metrics.recordedCount)
}

func TestCheckNodeEndpointErrors_EmptyGamesList(t *testing.T) {
	games := []*types.EnrichedGameData{}

	metrics := &stubNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	require.Equal(t, 0, metrics.recordedCount)
}

type stubNodeEndpointErrorsMetrics struct {
	recordedCount int
}

func (s *stubNodeEndpointErrorsMetrics) RecordNodeEndpointErrors(count int) {
	s.recordedCount = count
}
