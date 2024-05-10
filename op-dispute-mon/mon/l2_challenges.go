package mon

import "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"

type L2ChallengesMetrics interface {
	RecordL2Challenges(count int)
}

type L2ChallengesMonitor struct {
	metrics L2ChallengesMetrics
}

func NewL2ChallengesMonitor(metrics L2ChallengesMetrics) *L2ChallengesMonitor {
	return &L2ChallengesMonitor{
		metrics: metrics,
	}
}

func (m *L2ChallengesMonitor) CheckL2Challenges(games []*types.EnrichedGameData) {
	challengeCount := 0
	for _, game := range games {
		if game.BlockNumberChallenged {
			challengeCount++
		}
	}
	m.metrics.RecordL2Challenges(challengeCount)
}
