package monitor

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// SupernodeObserver watches one op-supernode (read-only). It records liveness,
// per-chain safe/finalized heads, and the highest-signal check: a bad executing
// message that the supernode has already promoted to cross-safe. It never gates
// monitor behaviour.
type SupernodeObserver struct {
	endpoint string
	client   SupernodeObserverClient
	m        InteropMessageMetrics
	log      log.Logger
	timeout  time.Duration
}

func NewSupernodeObserver(endpoint string, c SupernodeObserverClient, m InteropMessageMetrics, log log.Logger) *SupernodeObserver {
	return &SupernodeObserver{endpoint: endpoint, client: c, m: m, log: log, timeout: 2 * time.Second}
}

func (o *SupernodeObserver) Observe(ctx context.Context, jobs map[JobID]*Job) {
	cctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	if err := o.client.Heartbeat(cctx); err != nil {
		o.log.Error("supernode heartbeat failed", "endpoint", o.endpoint, "error", err)
		o.m.RecordSupernodeUp(o.endpoint, false)
		return
	}
	o.m.RecordSupernodeUp(o.endpoint, true)

	status, err := o.client.SyncStatus(cctx)
	if err != nil {
		o.log.Error("supernode syncStatus failed", "endpoint", o.endpoint, "error", err)
		return
	}
	for chainID, s := range status.Chains {
		// Post-interop, SafeL2 is the cross-safe head; FinalizedL2 is irreversible.
		o.m.RecordSupernodeSafeHead(chainID.String(), "cross_safe", s.SafeL2.Number)
		o.m.RecordSupernodeSafeHead(chainID.String(), "finalized", s.FinalizedL2.Number)
	}

	// Highest-signal check: a bad EM that the supernode already promoted to cross-safe.
	for _, job := range jobs {
		st := job.LatestStatus()
		if st == jobStatusValid || st == jobStatusUnknown {
			continue
		}
		s, ok := status.Chains[job.executingChain]
		if !ok {
			continue
		}
		if job.executingBlock.Number <= s.SafeL2.Number {
			o.log.Error("bad executing message at/below supernode cross-safe head",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block", job.executingBlock.Number,
				"cross_safe_head", s.SafeL2.Number,
				"status", st.String(),
			)
			o.m.RecordCrossSafetyViolation(job.executingChain.String(), job.initiating.ChainID.String(), "cross_safe")
		}
	}
}
