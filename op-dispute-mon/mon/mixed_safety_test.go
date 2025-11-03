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

func TestCheckMixedSafety(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, RollupEndpointSafeCount: 2, RollupEndpointUnsafeCount: 1},
		{RollupEndpointSafeCount: 3, RollupEndpointUnsafeCount: 0}, // All safe
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, RollupEndpointSafeCount: 1, RollupEndpointUnsafeCount: 4},
		{RollupEndpointSafeCount: 0, RollupEndpointUnsafeCount: 2}, // All unsafe
		{RollupEndpointSafeCount: 0, RollupEndpointUnsafeCount: 0}, // No safety checks
	}
	metrics := &stubMixedSafetyMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewMixedSafetyMonitor(logger, metrics)
	monitor.CheckMixedSafety(games)
	require.Equal(t, 2, metrics.recordedCount)

	// Debug log for first mixed safety game
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Mixed safety detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 2)

	l := logs[0]
	require.Equal(t, common.Address{0x11}, l.AttrValue("game"))
	require.Equal(t, int64(2), l.AttrValue("safeCount"))
	require.Equal(t, int64(1), l.AttrValue("unsafeCount"))

	// Info log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	messageFilter = testlog.NewMessageFilter("Mixed safety summary")
	l = capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(2), l.AttrValue("gamesWithMixedSafety"))
	require.Equal(t, int64(5), l.AttrValue("totalGames"))
}

type stubMixedSafetyMetrics struct {
	recordedCount int
}

func (s *stubMixedSafetyMetrics) RecordMixedSafetyGames(count int) {
	s.recordedCount = count
}
