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

func TestCheckDifferentOutputRoots(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointDifferentOutputRoots: true,
			L2BlockNumber:                      100,
			RootClaim:                          common.HexToHash("0xaaa"),
		},
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointDifferentOutputRoots: false, // No disagreement
			L2BlockNumber:                      200,
			RootClaim:                          common.HexToHash("0xbbb"),
		},
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointDifferentOutputRoots: true,
			L2BlockNumber:                      300,
			RootClaim:                          common.HexToHash("0xccc"),
		},
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x44}},
			RollupEndpointDifferentOutputRoots: false, // No disagreement
			L2BlockNumber:                      400,
			RootClaim:                          common.HexToHash("0xddd"),
		},
	}
	metrics := &stubDifferentOutputRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentOutputRootMonitor(logger, metrics)
	monitor.CheckDifferentOutputRoots(games)
	require.Equal(t, 2, metrics.recordedCount)

	// Debug log for first game with different output roots
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Different output roots detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 2)

	l := logs[0]
	require.Equal(t, common.Address{0x11}, l.AttrValue("game"))
	require.Equal(t, uint64(100), l.AttrValue("l2BlockNumber"))
	require.Equal(t, common.HexToHash("0xaaa"), l.AttrValue("rootClaim"))

	// Info log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	messageFilter = testlog.NewMessageFilter("Different output roots summary")
	l = capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(2), l.AttrValue("gamesWithDifferentOutputRoots"))
	require.Equal(t, int64(4), l.AttrValue("totalGames"))
}

func TestCheckDifferentOutputRoots_NoDisagreements(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointDifferentOutputRoots: false,
		},
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointDifferentOutputRoots: false,
		},
	}
	metrics := &stubDifferentOutputRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentOutputRootMonitor(logger, metrics)
	monitor.CheckDifferentOutputRoots(games)
	require.Equal(t, 0, metrics.recordedCount)

	// No info log should be present when count is 0
	levelFilter := testlog.NewLevelFilter(log.LevelInfo)
	messageFilter := testlog.NewMessageFilter("Different output roots summary")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.Nil(t, l)
}

func TestCheckDifferentOutputRoots_EmptyGamesList(t *testing.T) {
	games := []*types.EnrichedGameData{}
	metrics := &stubDifferentOutputRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentOutputRootMonitor(logger, metrics)
	monitor.CheckDifferentOutputRoots(games)
	require.Equal(t, 0, metrics.recordedCount)

	// No log should be present when no games exist
	levelFilter := testlog.NewLevelFilter(log.LevelInfo)
	messageFilter := testlog.NewMessageFilter("Different output roots summary")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.Nil(t, l)
}

func TestCheckDifferentOutputRoots_AllGamesHaveDisagreements(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			RollupEndpointDifferentOutputRoots: true,
			L2BlockNumber:                      100,
		},
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			RollupEndpointDifferentOutputRoots: true,
			L2BlockNumber:                      200,
		},
		{
			GameMetadata:                       gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			RollupEndpointDifferentOutputRoots: true,
			L2BlockNumber:                      300,
		},
	}
	metrics := &stubDifferentOutputRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentOutputRootMonitor(logger, metrics)
	monitor.CheckDifferentOutputRoots(games)
	require.Equal(t, 3, metrics.recordedCount)

	// Debug logs for all games
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Different output roots detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 3)

	// Info log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	messageFilter = testlog.NewMessageFilter("Different output roots summary")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(3), l.AttrValue("gamesWithDifferentOutputRoots"))
	require.Equal(t, int64(3), l.AttrValue("totalGames"))
}

type stubDifferentOutputRootMetrics struct {
	recordedCount int
}

func (s *stubDifferentOutputRootMetrics) RecordDifferentOutputRootGames(count int) {
	s.recordedCount = count
}
