package monitor

import (
	"context"
	"errors"
	"time"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
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
		// Query the filter only when the monitor verdict for this job is new.
		// Re-checking an unchanged status every cycle would issue one RPC per
		// in-flight terminal job per second; gating purely on "ever checked"
		// would miss divergences that appear after a status flip (e.g. an
		// initiating-chain reorg turning a valid job invalid).
		if job.FilterCheckedFor(status) {
			continue
		}
		msg := messages.Message{Identifier: *job.initiating, PayloadHash: job.executingPayload}
		cctx, cancel := context.WithTimeout(ctx, o.timeout)
		err := o.filter.CheckMessage(cctx, msg, job.executingChain, job.executingTimestamp)
		cancel()

		// Only a structured JSON-RPC response counts as a filter verdict. A transport
		// or timeout error is not a rejection: leave the job unmarked so it is retried
		// next cycle, rather than recording a false divergence.
		if err != nil {
			var rpcErr rpc.Error
			if !errors.As(err, &rpcErr) {
				o.log.Warn("interop-filter check failed (transport error); will retry",
					"executing_chain_id", job.executingChain,
					"initiating_chain_id", job.initiating.ChainID,
					"error", err,
				)
				continue
			}
		}
		// A verdict was returned; do not re-query this job until its status changes.
		job.MarkFilterChecked(status)

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
