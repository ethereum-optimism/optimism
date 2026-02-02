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

func TestCheckSuperNodeEndpointErrors_NoErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, SuperNodeEndpointErrors: nil},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, SuperNodeEndpointErrors: make(map[string]bool)},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}}}, // No SuperNodeEndpointErrors field set
	}

	metrics := &stubSuperNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrors(games)

	require.Equal(t, 0, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrors_SingleGameWithErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointErrors: map[string]bool{
				"endpoint_1": true,
				"endpoint_2": true,
			},
		},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, SuperNodeEndpointErrors: nil},
	}

	metrics := &stubSuperNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrors(games)

	require.Equal(t, 2, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrors_MultipleGamesWithOverlappingErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointErrors: map[string]bool{
				"endpoint_1": true,
				"endpoint_2": true,
			},
		},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointErrors: map[string]bool{
				"endpoint_2": true, // Overlapping with first game
				"endpoint_3": true,
			},
		},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			SuperNodeEndpointErrors: map[string]bool{
				"endpoint_4": true,
			},
		},
	}

	metrics := &stubSuperNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrors(games)

	// Should count unique endpoints across all games (endpoint_1, endpoint_2, endpoint_3, endpoint_4)
	require.Equal(t, 4, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrors_MixedGamesWithAndWithoutErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, SuperNodeEndpointErrors: nil},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointErrors: map[string]bool{
				"endpoint_1": true,
			},
		},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}}, SuperNodeEndpointErrors: make(map[string]bool)},
		{
			GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x44}},
			SuperNodeEndpointErrors: map[string]bool{
				"endpoint_2": true,
			},
		},
	}

	metrics := &stubSuperNodeEndpointErrorsMetrics{}
	logger, _ := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrors(games)

	require.Equal(t, 2, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrors_EmptyGamesList(t *testing.T) {
	games := []*types.EnrichedGameData{}

	metrics := &stubSuperNodeEndpointErrorsMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorsMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrors(games)

	require.Equal(t, 0, metrics.recordedCount)
}

type stubSuperNodeEndpointErrorsMetrics struct {
	recordedCount int
}

func (s *stubSuperNodeEndpointErrorsMetrics) RecordSuperNodeEndpointErrors(count int) {
	s.recordedCount = count
}
