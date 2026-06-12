package interop

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ErrResumeDivergence is returned when the resume consistency check finds that
// durable local state (logsDB, verifiedDB) no longer agrees with itself or
// with the chains, in a way the round loop's reorg machinery cannot explain
// or repair. See checkResumeConsistency for the case analysis.
var ErrResumeDivergence = errors.New("resume state diverged from chain")

const resumeRemediation = "if the environment was reset (devnet re-genesis, EL restore) wipe the supernode data dir; otherwise investigate L2 derivation divergence before restarting"

// checkResumeConsistency validates, once per process start on the
// verifiedDB-resume path, that the durable state the verifier is about to
// build on is still trustworthy — both internally (logsDB tips match the last
// verified heads) and externally (sealed tips are still canonical on their
// chains).
//
// Why this exists: the cold-start path self-heals offline divergence via
// reconcileLogsDBTail before backfilling, but the resume path historically
// went straight into the round loop. Protocol-level reorgs that happened
// while the supernode was offline are detected there — the
// lastVerified.L1Inclusion canonicality check fires and the loop rewinds — so
// chain divergence is only treated as terminal here when L1 still agrees with
// the verified history, a state no protocol reorg can produce (the L1 fork
// point of any reorg deep enough to change a verified block is at or below
// the recorded L1Inclusion). That state means the environment changed
// underneath us (EL restored from a snapshot, devnet re-genesis against a
// stale data dir) or derivation is non-deterministic. Proceeding would be
// unsound: a divergent logsDB fails cross-chain Contains checks with
// ErrConflict, which can wrongly invalidate legitimate blocks.
//
// Returns (0, nil) when the check passes or is deferred, (backoffPeriod, nil)
// while chains are still coming up, and a terminal error on divergence.
func (i *Interop) checkResumeConsistency() (time.Duration, error) {
	// A pending transition means the previous run stopped mid-apply; the WAL
	// replay path owns convergence from there (divergence during replay is
	// halt-classified in progress). The strict tip==head equality below does
	// not hold mid-transition, so the check must not run.
	pending, err := i.verifiedDB.GetPendingTransition()
	if err != nil {
		return 0, fmt.Errorf("resume consistency: read pending transition: %w", err)
	}
	if pending != nil {
		i.log.Info("resume consistency check deferred to WAL replay",
			"pendingDecision", pending.Decision)
		i.resumeChecked = true
		return 0, nil
	}

	lastTS, ok := i.verifiedDB.LastTimestamp()
	if !ok {
		// Nothing verified yet; nothing to validate. (The cold-start path
		// normally owns this state, so this is just a safe fallback.)
		i.resumeChecked = true
		return 0, nil
	}
	lastResult, err := i.verifiedDB.Get(lastTS)
	if err != nil {
		return 0, fmt.Errorf("resume consistency: read verified result at %d: %w", lastTS, err)
	}

	// Local invariant first (no RPC): with no pending transition, every apply
	// path re-establishes logsDB-tip == verified-head before clearing the
	// WAL, so a mismatch here is data loss or a foreign data dir — never a
	// reorg. Chains absent from the last entry (added to the dependency set
	// since) have no local state to validate.
	for chainID, head := range lastResult.L2Heads {
		db, ok := i.logsDBs[chainID]
		if !ok {
			continue
		}
		tip, has := db.LatestSealedBlock()
		if !has {
			return 0, i.halt("resume_divergence", "logsDB empty for chain with verified history",
				fmt.Errorf("chain %s: verified head %v at ts=%d but logsDB has no sealed blocks: %w",
					chainID, head, lastTS, ErrResumeDivergence), resumeRemediation)
		}
		if tip != head {
			return 0, i.halt("resume_divergence", "logsDB tip does not match last verified head",
				fmt.Errorf("chain %s: verified head %v at ts=%d but logsDB tip is %v: %w",
					chainID, head, lastTS, tip, ErrResumeDivergence), resumeRemediation)
		}
	}

	// Chain check: each verified head must still be the canonical block at
	// its height. Chains come up concurrently with this loop, so every read
	// error is treated as transient, matching cold-start init.
	for chainID, head := range lastResult.L2Heads {
		chain, ok := i.chains[chainID]
		if !ok {
			continue
		}
		out, err := chain.OutputV0AtBlockNumber(i.ctx, head.Number)
		if err != nil {
			i.log.Debug("resume consistency: chain not ready, will retry",
				"chain", chainID, "err", err)
			return backoffPeriod, nil
		}
		if out.BlockHash == head.Hash {
			continue
		}
		// The verified head is no longer canonical. Distinguish an offline L1
		// reorg (round loop rewinds; proceed) from divergence under an
		// unchanged L1 (terminal).
		if i.l1Checker == nil {
			i.log.Warn("resume consistency: verified head diverged but no L1 checker configured; deferring to round loop",
				"chain", chainID, "verifiedHead", head, "canonical", out.BlockHash)
			continue
		}
		same, err := i.l1Checker.SameL1Chain(i.ctx, []eth.BlockID{lastResult.L1Inclusion})
		if err != nil {
			i.log.Debug("resume consistency: L1 check failed, will retry",
				"chain", chainID, "err", err)
			return backoffPeriod, nil
		}
		if !same {
			i.log.Warn("resume consistency: verified head diverged due to offline L1 reorg; round loop will rewind",
				"chain", chainID, "verifiedHead", head, "canonical", out.BlockHash)
			continue
		}
		return 0, i.halt("resume_divergence", "verified chain state diverged while L1 inclusion is still canonical",
			fmt.Errorf("chain %s: verified head %v at ts=%d but canonical block at that height is %s: %w",
				chainID, head, lastTS, out.BlockHash, ErrResumeDivergence), resumeRemediation)
	}

	i.resumeChecked = true
	i.log.Info("resume consistency check passed",
		"lastVerifiedTimestamp", lastTS, "chains", len(lastResult.L2Heads))
	return 0, nil
}
