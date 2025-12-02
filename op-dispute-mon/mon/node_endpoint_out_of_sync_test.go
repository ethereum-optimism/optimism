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

func TestCheckNodeEndpointOutOfSync_NoOutOfSync(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, RollupEndpointOutOfSyncCount: 0},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, RollupEndpointOutOfSyncCount: 0},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x33}}, RollupEndpointOutOfSyncCount: 0},
	}

	metrics := &stubNodeEndpointOutOfSyncMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointOutOfSyncMonitor(logger, metrics)

	monitor.CheckNodeEndpointOutOfSync(games)

	require.Equal(t, 0, metrics.recordedCount)
}

func TestCheckNodeEndpointOutOfSync_SingleGameOutOfSync(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointOutOfSyncCount: 5,
		},
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointOutOfSyncCount: 0,
		},
	}

	metrics := &stubNodeEndpointOutOfSyncMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointOutOfSyncMonitor(logger, metrics)

	monitor.CheckNodeEndpointOutOfSync(games)

	require.Equal(t, 5, metrics.recordedCount)
}

func TestCheckNodeEndpointOutOfSync_MultipleGamesOutOfSync(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointOutOfSyncCount: 3,
		},
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointOutOfSyncCount: 7,
		},
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointOutOfSyncCount: 2,
		},
	}

	metrics := &stubNodeEndpointOutOfSyncMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointOutOfSyncMonitor(logger, metrics)

	monitor.CheckNodeEndpointOutOfSync(games)

	// Should sum all out-of-sync counts (3 + 7 + 2 = 12)
	require.Equal(t, 12, metrics.recordedCount)
}

func TestCheckNodeEndpointOutOfSync_MixedGamesWithAndWithoutOutOfSync(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointOutOfSyncCount: 0,
		},
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointOutOfSyncCount: 4,
		},
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointOutOfSyncCount: 0,
		},
		{
			GameMetadata:                 gameTypes.GameMetadata{Proxy: common.Address{0x44}},
			RollupEndpointOutOfSyncCount: 6,
		},
	}

	metrics := &stubNodeEndpointOutOfSyncMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointOutOfSyncMonitor(logger, metrics)

	monitor.CheckNodeEndpointOutOfSync(games)

	// Should sum only non-zero out-of-sync counts (4 + 6 = 10)
	require.Equal(t, 10, metrics.recordedCount)
}

func TestCheckNodeEndpointOutOfSync_EmptyGamesList(t *testing.T) {
	games := []*types.EnrichedGameData{}

	metrics := &stubNodeEndpointOutOfSyncMetrics{}
	logger := testlog.Logger(t, log.LvlDebug)
	monitor := NewNodeEndpointOutOfSyncMonitor(logger, metrics)

	monitor.CheckNodeEndpointOutOfSync(games)

	require.Equal(t, 0, metrics.recordedCount)
}

type stubNodeEndpointOutOfSyncMetrics struct {
	recordedCount int
}

func (s *stubNodeEndpointOutOfSyncMetrics) RecordNodeEndpointOutOfSyncCount(count int) {
	s.recordedCount = count
}
