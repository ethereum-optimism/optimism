package mon

import (
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

const MaxResolveDelay = time.Minute

type ResolutionMetrics interface {
	RecordGameResolutionStatus(status metrics.ResolutionStatus, count int)
}

type ResolutionMonitor struct {
	logger  log.Logger
	clock   RClock
	metrics ResolutionMetrics
}

func NewResolutionMonitor(logger log.Logger, metrics ResolutionMetrics, clock RClock) *ResolutionMonitor {
	return &ResolutionMonitor{
		logger:  logger,
		clock:   clock,
		metrics: metrics,
	}
}

func (r *ResolutionMonitor) CheckResolutions(games []*types.EnrichedGameData) {
	statusMetrics := make(map[metrics.ResolutionStatus]int)
	for _, game := range games {
		complete := game.Status != gameTypes.GameStatusInProgress
		duration := uint64(r.clock.Now().Unix()) - game.Timestamp
		maxDurationReached := duration >= (2 * game.MaxClockDuration)
		resolvable := true
		for _, claim := range game.Claims {
			// If any claim is not resolved, the game is not resolvable
			resolvable = resolvable && claim.Resolved
		}
		if complete {
			if maxDurationReached {
				statusMetrics[metrics.CompleteMaxDuration]++
			} else {
				statusMetrics[metrics.CompleteBeforeMaxDuration]++
			}
		} else if resolvable {
			if maxDurationReached {
				// SAFETY: since maxDurationReached is true, this cannot underflow
				delay := duration - (2 * game.MaxClockDuration)
				if delay > uint64(MaxResolveDelay.Seconds()) {
					r.logger.Warn("Resolvable game has taken too long to resolve", "game", game.Proxy, "delay", delay)
				}
				statusMetrics[metrics.ResolvableMaxDuration]++
			} else {
				statusMetrics[metrics.ResolvableBeforeMaxDuration]++
			}
		} else {
			if maxDurationReached {
				// Note: we don't need to log here since unresolved claims are logged and metriced in claims.go
				statusMetrics[metrics.InProgressMaxDuration]++
			} else {
				statusMetrics[metrics.InProgressBeforeMaxDuration]++
			}
		}
	}

	r.metrics.RecordGameResolutionStatus(metrics.CompleteMaxDuration, statusMetrics[metrics.CompleteMaxDuration])
	r.metrics.RecordGameResolutionStatus(metrics.CompleteBeforeMaxDuration, statusMetrics[metrics.CompleteBeforeMaxDuration])
	r.metrics.RecordGameResolutionStatus(metrics.ResolvableMaxDuration, statusMetrics[metrics.ResolvableMaxDuration])
	r.metrics.RecordGameResolutionStatus(metrics.ResolvableBeforeMaxDuration, statusMetrics[metrics.ResolvableBeforeMaxDuration])
	r.metrics.RecordGameResolutionStatus(metrics.InProgressMaxDuration, statusMetrics[metrics.InProgressMaxDuration])
	r.metrics.RecordGameResolutionStatus(metrics.InProgressBeforeMaxDuration, statusMetrics[metrics.InProgressBeforeMaxDuration])
}
