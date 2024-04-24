package mon

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type ResolutionMetrics interface {
	RecordGameResolutionStatus(complete bool, maxDurationReached bool, count int)
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

type resolutionStatus struct {
	completeMaxDuration         int
	completeBeforeMaxDuration   int
	inProgressMaxDuration       int
	inProgressBeforeMaxDuration int
}

func (r *resolutionStatus) Inc(complete, maxDuration bool) {
	if complete {
		if maxDuration {
			r.completeMaxDuration++
		} else {
			r.completeBeforeMaxDuration++
		}
	} else {
		if maxDuration {
			r.inProgressMaxDuration++
		} else {
			r.inProgressBeforeMaxDuration++
		}
	}
}

func (r *ResolutionMonitor) CheckResolutions(games []*types.EnrichedGameData) {
	status := &resolutionStatus{}
	for _, game := range games {
		complete := game.Status != gameTypes.GameStatusInProgress
		duration := uint64(r.clock.Now().Unix()) - game.Timestamp
		maxDurationReached := duration >= game.MaxClockDuration
		status.Inc(complete, maxDurationReached)
	}
	r.metrics.RecordGameResolutionStatus(true, true, status.completeMaxDuration)
	r.metrics.RecordGameResolutionStatus(true, false, status.completeBeforeMaxDuration)
	r.metrics.RecordGameResolutionStatus(false, true, status.inProgressMaxDuration)
	r.metrics.RecordGameResolutionStatus(false, false, status.inProgressBeforeMaxDuration)
}
