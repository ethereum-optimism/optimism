package interop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// ErrChainBehindBackfillLowerBound is returned when a chain's LocalSafeL2
// block number is earlier than the block that TimestampToBlockNumber maps
// T_lo to. Under a consistent SyncStatus we should always have
// LocalSafeL2.Time >= minCrossSafeTime >= T_lo, so this indicates the
// chain container is reporting inconsistent state (e.g. mid-rewind, a
// fresh VN whose LocalSafe hasn't caught up to cross-safe, or a bad
// TimestampToBlockNumber implementation). Silently skipping such a
// chain would cause the main loop to wedge on ErrStaleLogsDB once it
// hands off below the first sealed block of chains that did backfill,
// so we fail the whole attempt instead. Operator remediation is to
// clear the logs DBs and restart.
var ErrChainBehindBackfillLowerBound = errors.New("chain localSafe is behind backfill lower bound; clear data dir")

// LogBackfillLowerBound returns T_lo = max(T_act, crossSafeTs - D_log) in unix seconds (L2).
// crossSafeTs is the minimum cross-safe timestamp across all chains — the
// earliest point where cross-validation will resume after startup.
// Never ingest logs for timestamps before activation.
func LogBackfillLowerBound(crossSafeTs, activationTimestampUnix uint64, logBackfillDepth time.Duration) uint64 {
	if logBackfillDepth <= 0 {
		return crossSafeTs
	}
	sub := uint64(logBackfillDepth / time.Second)
	var raw uint64
	if crossSafeTs >= sub {
		raw = crossSafeTs - sub
	} else {
		raw = 0
	}
	if raw < activationTimestampUnix {
		return activationTimestampUnix
	}
	return raw
}

// runLogBackfill seals logs for each chain from T_lo through LocalSafe and
// advances activationTimestamp past the backfilled range so the main loop
// starts verification after the pre-ingested data.
//
// T_lo is computed from the minimum cross-safe (SafeL2) timestamp across all
// chains, since that is the earliest point where cross-validation will resume.
func (i *Interop) runLogBackfill() error {
	if i.logBackfillDepth <= 0 {
		return nil
	}
	if len(i.chains) == 0 {
		return nil
	}

	ctx := i.ctx

	// First pass: gather the minimum cross-safe timestamp across all chains.
	// SafeL2 is the cross-safe head post-interop.
	type chainInfo struct {
		crossSafeTime uint64
		localSafeNum  uint64
		localSafeTime uint64
	}
	info := make(map[eth.ChainID]chainInfo, len(i.chains))
	var minCrossSafeTime uint64
	first := true
	for cid, chain := range i.chains {
		ss, err := chain.SyncStatus(ctx)
		if err != nil {
			return fmt.Errorf("chain %s: sync status: %w", cid, err)
		}
		ci := chainInfo{
			crossSafeTime: ss.SafeL2.Time,
			localSafeNum:  ss.LocalSafeL2.Number,
			localSafeTime: ss.LocalSafeL2.Time,
		}
		info[cid] = ci
		if first || ci.crossSafeTime < minCrossSafeTime {
			minCrossSafeTime = ci.crossSafeTime
			first = false
		}
	}

	Tlo := LogBackfillLowerBound(minCrossSafeTime, i.runtimeActivationTimestamp, i.logBackfillDepth)
	// Debug-level because this fires on every retry while VNs are coming up.
	// The summary "interop log backfill complete" line at the end is the
	// user-visible signal that backfill finished.
	i.log.Debug("log backfill: computed lower bound",
		"minCrossSafeTime", minCrossSafeTime, "T_lo", Tlo, "depth", i.logBackfillDepth)

	// Second pass: backfill each chain from T_lo to its LocalSafe. Fold
	// localSafeTime into minLocalSafeTime only after the consistency check
	// passes, so runtimeActivation is clamped to the earliest backfilled
	// head — any chain that can't satisfy the round aborts before
	// contributing, which keeps minLocalSafeTime reflective of the set of
	// chains whose logs DB is populated up to T_lo.
	var minLocalSafeTime uint64
	firstLocal := true
	for cid, chain := range i.chains {
		ci := info[cid]

		startNum, err := chain.TimestampToBlockNumber(ctx, Tlo)
		if err != nil {
			return fmt.Errorf("chain %s: timestamp to block number for T_lo %d: %w", cid, Tlo, err)
		}
		// Under a consistent SyncStatus this is unreachable: T_lo is clamped
		// to min(crossSafe) across chains, and localSafe >= crossSafe on every
		// chain, so localSafeNum >= TimestampToBlockNumber(T_lo). If it fires,
		// the chain container is reporting inconsistent state and continuing
		// would leave this chain's logs DB empty while other chains seal from
		// T_lo forward, which wedges the main loop's first seal attempt on
		// ErrStaleLogsDB.
		if startNum > ci.localSafeNum {
			return fmt.Errorf("chain %s: startNum %d > localSafeNum %d at T_lo %d (localSafeTime %d, minCrossSafeTime %d): %w",
				cid, startNum, ci.localSafeNum, Tlo, ci.localSafeTime, minCrossSafeTime, ErrChainBehindBackfillLowerBound)
		}

		if firstLocal || ci.localSafeTime < minLocalSafeTime {
			minLocalSafeTime = ci.localSafeTime
			firstLocal = false
		}

		i.log.Info("log backfill: sealing logs",
			"chain", cid, "from", startNum, "to", ci.localSafeNum)

		if err := i.backfillChain(ctx, cid, chain, startNum, ci.localSafeNum); err != nil {
			return err
		}
	}

	if !firstLocal && minLocalSafeTime+1 > i.runtimeActivationTimestamp {
		i.log.Info("advancing runtime activation past backfilled range",
			"oldActivation", i.runtimeActivationTimestamp, "newActivation", minLocalSafeTime+1)
		i.runtimeActivationTimestamp = minLocalSafeTime + 1
	}
	i.log.Info("interop log backfill complete", "runtimeActivationTimestamp", i.runtimeActivationTimestamp)
	return nil
}

func (i *Interop) backfillChain(ctx context.Context, cid eth.ChainID, chain cc.ChainContainer, startNum, endNum uint64) error {
	db := i.logsDBs[cid]
	if latest, has := db.LatestSealedBlock(); has {
		startNum = latest.Number + 1
	}
	for num := startNum; num <= endNum; num++ {
		out, err := chain.OutputV0AtBlockNumber(ctx, num)
		if err != nil {
			return fmt.Errorf("chain %s: output at block %d: %w", cid, num, err)
		}
		bid := eth.BlockID{Hash: out.BlockHash, Number: num}
		blockInfo, receipts, err := chain.FetchReceipts(ctx, bid)
		if err != nil {
			return fmt.Errorf("chain %s: fetch receipts %d: %w", cid, num, err)
		}

		if err := i.sealBlockDataIntoLogsDB(cid, bid, blockInfo, receipts, blockInfo.Time(), true); err != nil {
			return err
		}
	}
	return nil
}
