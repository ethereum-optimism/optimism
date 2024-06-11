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

func TestMonitorL2Challenges(t *testing.T) {
	games := []*types.EnrichedGameData{
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x44}}, BlockNumberChallenged: true, AgreeWithClaim: true, L2BlockNumber: 44, BlockNumberChallenger: common.Address{0x55}},
		{BlockNumberChallenged: false, AgreeWithClaim: true},
		{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0x22}}, BlockNumberChallenged: true, AgreeWithClaim: false, L2BlockNumber: 22, BlockNumberChallenger: common.Address{0x33}},
		{BlockNumberChallenged: false, AgreeWithClaim: false},
		{BlockNumberChallenged: false, AgreeWithClaim: false},
		{BlockNumberChallenged: false, AgreeWithClaim: true},
	}
	metrics := &stubL2ChallengeMetrics{}
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	monitor := NewL2ChallengesMonitor(logger, metrics)
	monitor.CheckL2Challenges(games)
	require.Equal(t, 1, metrics.challengeCount[true])
	require.Equal(t, 1, metrics.challengeCount[false])

	// Warn log for challenged and agreement
	levelFilter := testlog.NewLevelFilter(log.LevelWarn)
	messageFilter := testlog.NewMessageFilter("Found game with valid block number challenged")
	l := capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, common.Address{0x44}, l.AttrValue("game"))
	require.Equal(t, uint64(44), l.AttrValue("blockNum"))
	require.Equal(t, true, l.AttrValue("agreement"))
	require.Equal(t, common.Address{0x55}, l.AttrValue("challenger"))

	// Debug log for challenged but disagreement
	levelFilter = testlog.NewLevelFilter(log.LevelDebug)
	messageFilter = testlog.NewMessageFilter("Found game with invalid block number challenged")
	l = capturedLogs.FindLog(levelFilter, messageFilter)
	require.NotNil(t, l)
	require.Equal(t, common.Address{0x22}, l.AttrValue("game"))
	require.Equal(t, uint64(22), l.AttrValue("blockNum"))
	require.Equal(t, false, l.AttrValue("agreement"))
	require.Equal(t, common.Address{0x33}, l.AttrValue("challenger"))
}

type stubL2ChallengeMetrics struct {
	challengeCount map[bool]int
}

func (s *stubL2ChallengeMetrics) RecordL2Challenges(agreement bool, count int) {
	if s.challengeCount == nil {
		s.challengeCount = make(map[bool]int)
	}
	s.challengeCount[agreement] = count
}
