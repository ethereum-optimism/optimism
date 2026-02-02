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

func TestCheckMixedSuperSafety(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, SuperNodeEndpointSafeCount: 2, SuperNodeEndpointUnsafeCount: 1},
		{SuperNodeEndpointSafeCount: 3, SuperNodeEndpointUnsafeCount: 0}, // All safe
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, SuperNodeEndpointSafeCount: 1, SuperNodeEndpointUnsafeCount: 4},
		{SuperNodeEndpointSafeCount: 0, SuperNodeEndpointUnsafeCount: 2}, // All unsafe
		{SuperNodeEndpointSafeCount: 0, SuperNodeEndpointUnsafeCount: 0}, // No safety checks
	}
	metrics := &stubMixedSuperSafetyMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewMixedSuperSafetyMonitor(logger, metrics)
	monitor.CheckMixedSuperSafety(games)
	require.Equal(t, 2, metrics.recordedCount)

	// Debug log for first mixed super safety game
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Mixed super safety detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 2)

	l := logs[0]
	require.Equal(t, common.Address{0x11}, l.AttrValue("game"))
	require.Equal(t, int64(2), l.AttrValue("safeCount"))
	require.Equal(t, int64(1), l.AttrValue("unsafeCount"))

	// Info log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	messageFilter = testlog.NewMessageFilter("Mixed super safety summary")
	l = capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(2), l.AttrValue("gamesWithMixedSuperSafety"))
	require.Equal(t, int64(5), l.AttrValue("totalGames"))
}

type stubMixedSuperSafetyMetrics struct {
	recordedCount int
}

func (s *stubMixedSuperSafetyMetrics) RecordMixedSuperSafetyGames(count int) {
	s.recordedCount = count
}
