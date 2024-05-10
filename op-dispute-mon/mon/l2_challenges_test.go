package mon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/stretchr/testify/require"
)

func TestMonitorL2Challenges(t *testing.T) {
	games := []*types.EnrichedGameData{
		{BlockNumberChallenged: true},
		{BlockNumberChallenged: false},
		{BlockNumberChallenged: true},
		{BlockNumberChallenged: false},
		{BlockNumberChallenged: false},
		{BlockNumberChallenged: false},
	}
	metrics := &stubL2ChallengeMetrics{}
	monitor := NewL2ChallengesMonitor(metrics)
	monitor.CheckL2Challenges(games)
	require.Equal(t, 2, metrics.challengeCount)
}

type stubL2ChallengeMetrics struct {
	challengeCount int
}

func (s *stubL2ChallengeMetrics) RecordL2Challenges(count int) {
	s.challengeCount = count
}
