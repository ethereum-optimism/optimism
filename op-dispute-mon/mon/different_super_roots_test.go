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

func TestCheckDifferentSuperRoots(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointDifferentSuperRoots: true,
			L2SequenceNumber:                     100,
			RootClaim:                            common.HexToHash("0xaaa"),
		},
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointDifferentSuperRoots: false, // No disagreement
			L2SequenceNumber:                     200,
			RootClaim:                            common.HexToHash("0xbbb"),
		},
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			SuperNodeEndpointDifferentSuperRoots: true,
			L2SequenceNumber:                     300,
			RootClaim:                            common.HexToHash("0xccc"),
		},
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x44}},
			SuperNodeEndpointDifferentSuperRoots: false, // No disagreement
			L2SequenceNumber:                     400,
			RootClaim:                            common.HexToHash("0xddd"),
		},
	}
	metrics := &stubDifferentSuperRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentSuperRootMonitor(logger, metrics)
	monitor.CheckDifferentSuperRoots(games)
	require.Equal(t, 2, metrics.recordedCount)

	// Debug log for first game with different super roots
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Different super roots detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 2)

	l := logs[0]
	require.Equal(t, common.Address{0x11}, l.AttrValue("game"))
	require.Equal(t, uint64(100), l.AttrValue("l2SequenceNumber"))
	require.Equal(t, common.HexToHash("0xaaa"), l.AttrValue("rootClaim"))

	// Info log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	messageFilter = testlog.NewMessageFilter("Different super roots summary")
	l = capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(2), l.AttrValue("gamesWithDifferentSuperRoots"))
	require.Equal(t, int64(4), l.AttrValue("totalGames"))
}

func TestCheckDifferentSuperRoots_NoDisagreements(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointDifferentSuperRoots: false,
		},
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointDifferentSuperRoots: false,
		},
	}
	metrics := &stubDifferentSuperRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentSuperRootMonitor(logger, metrics)
	monitor.CheckDifferentSuperRoots(games)
	require.Equal(t, 0, metrics.recordedCount)

	// No info log should be present when count is 0
	levelFilter := testlog.NewLevelFilter(log.LevelInfo)
	messageFilter := testlog.NewMessageFilter("Different super roots summary")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.Nil(t, l)
}

func TestCheckDifferentSuperRoots_EmptyGamesList(t *testing.T) {
	games := []*types.EnrichedGameData{}
	metrics := &stubDifferentSuperRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentSuperRootMonitor(logger, metrics)
	monitor.CheckDifferentSuperRoots(games)
	require.Equal(t, 0, metrics.recordedCount)

	// No log should be present when no games exist
	levelFilter := testlog.NewLevelFilter(log.LevelInfo)
	messageFilter := testlog.NewMessageFilter("Different super roots summary")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.Nil(t, l)
}

func TestCheckDifferentSuperRoots_AllGamesHaveDisagreements(t *testing.T) {
	games := []*types.EnrichedGameData{
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x11}},
			SuperNodeEndpointDifferentSuperRoots: true,
			L2SequenceNumber:                     100,
		},
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x22}},
			SuperNodeEndpointDifferentSuperRoots: true,
			L2SequenceNumber:                     200,
		},
		{
			GameMetadata:                         gameTypes.GameMetadata{Proxy: common.Address{0x33}},
			SuperNodeEndpointDifferentSuperRoots: true,
			L2SequenceNumber:                     300,
		},
	}
	metrics := &stubDifferentSuperRootMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewDifferentSuperRootMonitor(logger, metrics)
	monitor.CheckDifferentSuperRoots(games)
	require.Equal(t, 3, metrics.recordedCount)

	// Debug logs for all games
	levelFilter := testlog.NewLevelFilter(log.LevelDebug)
	messageFilter := testlog.NewMessageFilter("Different super roots detected")
	logs := capturedLogs.FindLogs(levelFilter, messageFilter)
	require.Len(t, logs, 3)

	// Info log for summary
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	messageFilter = testlog.NewMessageFilter("Different super roots summary")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, int64(3), l.AttrValue("gamesWithDifferentSuperRoots"))
	require.Equal(t, int64(3), l.AttrValue("totalGames"))
}

type stubDifferentSuperRootMetrics struct {
	recordedCount int
}

func (s *stubDifferentSuperRootMetrics) RecordDifferentSuperRootGames(count int) {
	s.recordedCount = count
}
