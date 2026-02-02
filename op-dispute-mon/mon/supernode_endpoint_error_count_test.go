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

func TestCheckSuperNodeEndpointErrorCount_NoErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, SuperNodeEndpointErrorCount: 0},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, SuperNodeEndpointErrorCount: 0},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}}, SuperNodeEndpointErrorCount: 0},
	}

	metrics := &stubSuperNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrorCount(games)

	require.Equal(t, 0, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrorCount_SingleGameWithErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointErrorCount: 5,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointErrorCount: 0,
		},
	}

	metrics := &stubSuperNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrorCount(games)

	require.Equal(t, 5, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrorCount_MultipleGamesWithErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointErrorCount: 3,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointErrorCount: 7,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			SuperNodeEndpointErrorCount: 2,
		},
	}

	metrics := &stubSuperNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrorCount(games)

	// Should sum all error counts (3 + 7 + 2 = 12)
	require.Equal(t, 12, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrorCount_MixedGamesWithAndWithoutErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointErrorCount: 0,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointErrorCount: 4,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			SuperNodeEndpointErrorCount: 0,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x44}},
			SuperNodeEndpointErrorCount: 6,
		},
	}

	metrics := &stubSuperNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrorCount(games)

	// Should sum only non-zero error counts (4 + 6 = 10)
	require.Equal(t, 10, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrorCount_EmptyGamesList(t *testing.T) {
	games := []*types.EnrichedGameData{}

	metrics := &stubSuperNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrorCount(games)

	require.Equal(t, 0, metrics.recordedCount)
}

func TestCheckSuperNodeEndpointErrorCount_HighVolumeErrors(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointErrorCount: 100,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointErrorCount: 250,
		},
		{
			GameMetadata:                gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			SuperNodeEndpointErrorCount: 75,
		},
	}

	metrics := &stubSuperNodeEndpointErrorCountMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewSuperNodeEndpointErrorCountMonitor(logger, metrics)

	monitor.CheckSuperNodeEndpointErrorCount(games)

	// Should sum all error counts (100 + 250 + 75 = 425)
	require.Equal(t, 425, metrics.recordedCount)
}

type stubSuperNodeEndpointErrorCountMetrics struct {
	recordedCount int
}

func (s *stubSuperNodeEndpointErrorCountMetrics) RecordSuperNodeEndpointErrorCount(count int) {
	s.recordedCount = count
}
