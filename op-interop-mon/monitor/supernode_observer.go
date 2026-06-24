package monitor

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// SupernodeObserverClient is the read-only op-supernode surface the observer needs.
// It is satisfied by op-service/sources.SuperNodeClient.
type SupernodeObserverClient interface {
	// SyncStatus returns the supernode's aggregate per-chain sync status. A successful
	// call is also taken as the supernode's liveness signal.
	SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error)
	Close()
}

// CanonicalBlockSource is the minimal execution-layer surface used to confirm a job's
// executing block is still canonical on its chain. Satisfied by sources.EthClient.
type CanonicalBlockSource interface {
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
}

// SupernodeObserver watches one op-supernode (read-only). It records liveness,
// per-chain safe/finalized heads, and the highest-signal check: a bad executing
// message that the supernode has already promoted to cross-safe. It never gates
// monitor behaviour.
type SupernodeObserver struct {
	endpoint string
	client   SupernodeObserverClient
	els      map[eth.ChainID]CanonicalBlockSource
	m        InteropMessageMetrics
	log      log.Logger
	timeout  time.Duration
}

func NewSupernodeObserver(endpoint string, c SupernodeObserverClient, els map[eth.ChainID]CanonicalBlockSource, m InteropMessageMetrics, log log.Logger) *SupernodeObserver {
	return &SupernodeObserver{endpoint: endpoint, client: c, els: els, m: m, log: log, timeout: 2 * time.Second}
}

func (o *SupernodeObserver) Observe(ctx context.Context, jobs map[JobID]*Job) {
	// A successful syncStatus call doubles as the supernode liveness probe.
	cctx, cancel := context.WithTimeout(ctx, o.timeout)
	status, err := o.client.SyncStatus(cctx)
	cancel()
	if err != nil {
		o.log.Error("supernode syncStatus failed", "endpoint", o.endpoint, "error", err)
		o.m.RecordSupernodeUp(o.endpoint, false)
		return
	}
	o.m.RecordSupernodeUp(o.endpoint, true)

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
		if job.executingBlock.Number > s.SafeL2.Number {
			continue
		}
		// Already flagged: don't re-confirm (and re-fetch the canonical block)
		// every cycle for a job whose violation has been counted.
		if job.ViolationCounted() {
			continue
		}
		// Jobs are keyed by executing block number, not hash. During a reorg an
		// orphaned block lingers in a bad status until finalization, while the
		// supernode's cross-safe chain holds the replacement block at that height.
		// Confirm the supernode actually cross-validated THIS block (by hash) before
		// flagging a violation, otherwise the metric false-positives on reorgs.
		el, ok := o.els[job.executingChain]
		if !ok {
			o.log.Warn("no execution client for executing chain; skipping cross-safety check",
				"executing_chain_id", job.executingChain)
			continue
		}
		ictx, icancel := context.WithTimeout(ctx, o.timeout)
		info, err := el.InfoByNumber(ictx, job.executingBlock.Number)
		icancel()
		if err != nil {
			o.log.Warn("failed to fetch canonical block; skipping cross-safety check",
				"executing_chain_id", job.executingChain,
				"executing_block", job.executingBlock.Number,
				"error", err,
			)
			continue
		}
		if info.Hash() != job.executingBlock.Hash {
			// The executing block was reorged out; the supernode validated the
			// replacement block at this height, not this orphaned one.
			continue
		}
		// Record each violating job once, not on every collection cycle.
		if job.CountViolationOnce() {
			o.log.Error("bad executing message at/below supernode cross-safe head",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block", job.executingBlock.Number,
				"cross_safe_head", s.SafeL2.Number,
				"status", st.String(),
			)
			o.m.RecordCrossSafetyViolation(job.executingChain.String(), job.initiating.ChainID.String())
		}
	}
}
