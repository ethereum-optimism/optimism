package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type L2ChallengesMetrics interface {
	RecordL2Challenges(agreement bool, count int)
}

type L2ChallengesMonitor struct {
	logger  log.Logger
	metrics L2ChallengesMetrics
}

func NewL2ChallengesMonitor(logger log.Logger, metrics L2ChallengesMetrics) *L2ChallengesMonitor {
	return &L2ChallengesMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *L2ChallengesMonitor) CheckL2Challenges(games []*types.EnrichedGameData) {
	agreeChallengeCount := 0
	disagreeChallengeCount := 0
	for _, game := range games {
		if game.BlockNumberChallenged {
			if game.AgreeWithClaim {
				m.logger.Warn("Found game with valid block number challenged",
					"game", game.Proxy, "blockNum", game.L2BlockNumber, "agreement", game.AgreeWithClaim, "challenger", game.BlockNumberChallenger)
				agreeChallengeCount++
			} else {
				m.logger.Debug("Found game with invalid block number challenged",
					"game", game.Proxy, "blockNum", game.L2BlockNumber, "agreement", game.AgreeWithClaim, "challenger", game.BlockNumberChallenger)
				disagreeChallengeCount++
			}
		}
	}
	m.metrics.RecordL2Challenges(true, agreeChallengeCount)
	m.metrics.RecordL2Challenges(false, disagreeChallengeCount)
}
