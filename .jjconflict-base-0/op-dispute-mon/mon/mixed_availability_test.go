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

func TestCheckMixedAvailability(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x11}}, RollupEndpointTotalCount: 5, RollupEndpointNotFoundCount: 2, RollupEndpointErrorCount: 1}, // Mixed (2 successful)
		{RollupEndpointTotalCount: 3, RollupEndpointNotFoundCount: 0, RollupEndpointErrorCount: 0},                                                                    // All successful
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, RollupEndpointTotalCount: 6, RollupEndpointNotFoundCount: 2, RollupEndpointErrorCount: 2}, // Mixed (2 successful)
		{RollupEndpointTotalCount: 3, RollupEndpointNotFoundCount: 3, RollupEndpointErrorCount: 0},                                                                    // All not found
		{RollupEndpointTotalCount: 2, RollupEndpointNotFoundCount: 0, RollupEndpointErrorCount: 2},                                                                    // All errors
	}
	metrics := &stubMixedAvailabilityMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewMixedAvailability(logger, metrics)
	monitor.CheckMixedAvailability(games)
	require.Equal(t, 2, metrics.recordedCount)

	// Debug log for first mixed availability game
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Mixed availability detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 2)

	l := logs[0]
	require.Equal(t, common.Address{0x11}, l.AttrValue("game"))
	require.Equal(t, int64(5), l.AttrValue("totalEndpoints"))
	require.Equal(t, int64(2), l.AttrValue("notFoundCount"))
	require.Equal(t, int64(1), l.AttrValue("errorCount"))

	// Warn log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelWarn)
	messageFilter = testlog.NewMessageFilter("Mixed availability summary")
	l = capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(2), l.AttrValue("gamesWithMixedAvailability"))
	require.Equal(t, int64(5), l.AttrValue("totalGames"))
}

type stubMixedAvailabilityMetrics struct {
	recordedCount int
}

func (s *stubMixedAvailabilityMetrics) RecordMixedAvailabilityGames(count int) {
	s.recordedCount = count
}
