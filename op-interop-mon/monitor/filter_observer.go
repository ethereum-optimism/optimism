package monitor

import (
	"context"
	"time"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum/go-ethereum/log"
)

// FilterObserver cross-checks the monitor's independent verdict against the
// op-interop-filter (read-only). It never gates monitor behaviour; it only emits
// divergence and failsafe metrics for observability.
type FilterObserver struct {
	filter  FilterChecker
	m       InteropMessageMetrics
	log     log.Logger
	timeout time.Duration
}

func NewFilterObserver(filter FilterChecker, m InteropMessageMetrics, log log.Logger) *FilterObserver {
	return &FilterObserver{filter: filter, m: m, log: log, timeout: 2 * time.Second}
}

// Observe replays each terminal job's executing message to the filter and records divergences.
func (o *FilterObserver) Observe(ctx context.Context, jobs map[JobID]*Job) {
	for _, job := range jobs {
		status := job.LatestStatus()
		// Only compare decided verdicts; unknown jobs are not yet resolved.
		if !status.isTerminal() {
			continue
		}
		msg := messages.Message{Identifier: *job.initiating, PayloadHash: job.executingPayload}
		cctx, cancel := context.WithTimeout(ctx, o.timeout)
		err := o.filter.CheckMessage(cctx, msg, job.executingChain, job.executingTimestamp)
		cancel()

		monitorValid := status == jobStatusValid
		filterValid := err == nil
		if monitorValid != filterValid {
			filterStatus := "valid"
			if !filterValid {
				filterStatus = "invalid"
			}
			o.log.Warn("monitor/filter verdict divergence",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"monitor_status", status.String(),
				"filter_status", filterStatus,
				"filter_err", err,
			)
			o.m.RecordFilterDivergence(job.executingChain.String(), job.initiating.ChainID.String(), status.String(), filterStatus)
		}
	}
}

// PollFailsafe reads the filter's failsafe state and records it as a gauge.
func (o *FilterObserver) PollFailsafe(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	enabled, err := o.filter.GetFailsafeEnabled(cctx)
	if err != nil {
		o.log.Error("failed to read interop-filter failsafe state", "error", err)
		return
	}
	o.m.RecordFilterFailsafe(enabled)
}
