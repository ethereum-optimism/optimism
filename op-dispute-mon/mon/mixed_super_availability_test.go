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

func TestCheckMixedSuperAvailability(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, SuperNodeEndpointTotalCount: 5, SuperNodeEndpointNotFoundCount: 2, SuperNodeEndpointErrorCount: 1}, // Mixed (2 successful)
		{SuperNodeEndpointTotalCount: 3, SuperNodeEndpointNotFoundCount: 0, SuperNodeEndpointErrorCount: 0},                                                                    // All successful
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, SuperNodeEndpointTotalCount: 6, SuperNodeEndpointNotFoundCount: 2, SuperNodeEndpointErrorCount: 2}, // Mixed (2 successful)
		{SuperNodeEndpointTotalCount: 3, SuperNodeEndpointNotFoundCount: 3, SuperNodeEndpointErrorCount: 0},                                                                    // All not found
		{SuperNodeEndpointTotalCount: 2, SuperNodeEndpointNotFoundCount: 0, SuperNodeEndpointErrorCount: 2},                                                                    // All errors
	}
	metrics := &stubMixedSuperAvailabilityMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewMixedSuperAvailability(logger, metrics)
	monitor.CheckMixedSuperAvailability(games)
	require.Equal(t, 2, metrics.recordedCount)

	// Debug log for first mixed super availability game
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Mixed super availability detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 2)

	l := logs[0]
	require.Equal(t, common.Address{0x11}, l.AttrValue("game"))
	require.Equal(t, int64(5), l.AttrValue("totalEndpoints"))
	require.Equal(t, int64(2), l.AttrValue("notFoundCount"))
	require.Equal(t, int64(1), l.AttrValue("errorCount"))

	// Info log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	messageFilter = testlog.NewMessageFilter("Mixed super availability summary")
	l = capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(2), l.AttrValue("gamesWithMixedSuperAvailability"))
	require.Equal(t, int64(5), l.AttrValue("totalGames"))
}

type stubMixedSuperAvailabilityMetrics struct {
	recordedCount int
}

func (s *stubMixedSuperAvailabilityMetrics) RecordMixedSuperAvailabilityGames(count int) {
	s.recordedCount = count
}
