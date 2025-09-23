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
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	require.Equal(t, 0, metrics.recordedCount)

	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("No rollup node endpoint errors found")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
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
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	require.Equal(t, 2, metrics.recordedCount)

	levelFilter := testlog.NewLevelFilter(log.LevelWarn)
	messageFilter := testlog.NewMessageFilter("Found rollup node endpoint errors")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(2), l.AttrValue("unique_endpoint_count"))
	endpoints := l.AttrValue("endpoints").([]string)
	require.Len(t, endpoints, 2)
	require.Contains(t, endpoints, "endpoint_1")
	require.Contains(t, endpoints, "endpoint_2")
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
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	// Should count unique endpoints across all games (endpoint_1, endpoint_2, endpoint_3, endpoint_4)
	require.Equal(t, 4, metrics.recordedCount)

	// Check warn log for errors found
	levelFilter := testlog.NewLevelFilter(log.LevelWarn)
	messageFilter := testlog.NewMessageFilter("Found rollup node endpoint errors")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(4), l.AttrValue("unique_endpoint_count"))
	endpoints := l.AttrValue("endpoints").([]string)
	require.Len(t, endpoints, 4)
	require.Contains(t, endpoints, "endpoint_1")
	require.Contains(t, endpoints, "endpoint_2")
	require.Contains(t, endpoints, "endpoint_3")
	require.Contains(t, endpoints, "endpoint_4")
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
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckNodeEndpointErrors(games)

	require.Equal(t, 0, metrics.recordedCount)

	// Check debug log for no errors
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("No rollup node endpoint errors found")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
}

type stubNodeEndpointErrorsMetrics struct {
	recordedCount int
}

func (s *stubNodeEndpointErrorsMetrics) RecordNodeEndpointErrors(count int) {
	s.recordedCount = count
}
