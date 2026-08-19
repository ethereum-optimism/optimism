package mon

import (
	"math"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
)

type ZKLifecycleMetrics interface {
	RecordZKGamesPendingLifecycleActions(resolution, bondDistribution int)
}

type ZKLifecycleMonitor struct {
	clock   RClock
	metrics ZKLifecycleMetrics
}

func NewZKLifecycleMonitor(clock RClock, metrics ZKLifecycleMetrics) *ZKLifecycleMonitor {
	return &ZKLifecycleMonitor{clock: clock, metrics: metrics}
}

func (m *ZKLifecycleMonitor) CheckLifecycle(games []*types.ZKGameData) {
	pendingResolution := 0
	pendingBondDistribution := 0
	for _, game := range games {
		if m.needsResolution(game) {
			pendingResolution++
		}
		if game.Status != gameTypes.GameStatusInProgress &&
			game.BondDistributionMode == faultTypes.UndecidedDistributionMode {
			pendingBondDistribution++
		}
	}
	m.metrics.RecordZKGamesPendingLifecycleActions(pendingResolution, pendingBondDistribution)
}

func (m *ZKLifecycleMonitor) needsResolution(game *types.ZKGameData) bool {
	if game.Status != gameTypes.GameStatusInProgress || game.ProposalStatus == contracts.ProposalStatusResolved {
		return false
	}
	if game.ParentIndex != math.MaxUint32 {
		if game.ParentStatus == nil || *game.ParentStatus == gameTypes.GameStatusInProgress {
			return false
		}
		if *game.ParentStatus == gameTypes.GameStatusChallengerWon {
			return true
		}
	}
	if game.ProposalStatus == contracts.ProposalStatusChallengedAndValidProofProvided ||
		game.ProposalStatus == contracts.ProposalStatusUnchallengedAndValidProofProvided {
		return true
	}
	return game.Deadline.Before(m.clock.Now())
}
