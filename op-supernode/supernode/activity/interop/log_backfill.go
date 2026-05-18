package interop

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// resolveFirstVerifiableTimestamp returns the first timestamp not yet covered
// by durable local state: verifiedDB.LastTimestamp+1 when initialized,
// otherwise the minimum EL finalized head + 1 (clamped to activation).
func (i *Interop) resolveFirstVerifiableTimestamp(ctx context.Context) (uint64, error) {
	if len(i.chains) == 0 {
		return i.activationTimestamp, nil
	}
	if i.verifiedDB != nil {
		if lastTS, initialized := i.verifiedDB.LastTimestamp(); initialized {
			return lastTS + 1, nil
		}
	}
	minELFinalizedTime, err := i.minELFinalizedTime(ctx)
	if err != nil {
		return 0, err
	}
	if minELFinalizedTime < i.activationTimestamp {
		return i.activationTimestamp, nil
	}
	return minELFinalizedTime + 1, nil
}

func (i *Interop) minELFinalizedTime(ctx context.Context) (uint64, error) {
	if len(i.chains) == 0 {
		return i.activationTimestamp, nil
	}

	minELFinalizedTime := uint64(math.MaxUint64)
	for _, chain := range i.chains {
		elFinalized, err := chain.ELFinalizedHead(ctx)
		if err != nil {
			return 0, fmt.Errorf("chain %s: EL finalized head: %w", chain.ID(), err)
		}
		// Genesis (Number == 0) with a real hash is a legitimate finalized head;
		// only reject the zero-value response from an EL that isn't ready yet.
		if elFinalized == (eth.L2BlockRef{}) {
			return 0, fmt.Errorf("chain %s: EL finalized head not yet available", chain.ID())
		}
		i.log.Debug("first verifiable timestamp: EL finalized head",
			"chain", chain.ID(), "elFinalized", elFinalized)
		minELFinalizedTime = min(minELFinalizedTime, elFinalized.Time)
	}
	return minELFinalizedTime, nil
}

func (i *Interop) runLogBackfill() (uint64, error) {
	if i.logBackfillDepth <= 0 {
		return 0, nil
	}
	if len(i.chains) == 0 {
		return 0, nil
	}

	firstVerifiable := i.firstVerifiable
	if !i.firstVerifiableSet {
		var err error
		firstVerifiable, err = i.resolveFirstVerifiableTimestamp(i.ctx)
		if err != nil {
			return 0, err
		}
	}
	if firstVerifiable == i.activationTimestamp {
		return 0, nil
	}
	endTime := firstVerifiable - 1

	// naively, end minus depth is the ideal backfill start.
	// guard the subtraction so a young chain (EL finalized < depth) doesn't wrap.
	depthSec := uint64(i.logBackfillDepth.Seconds())
	var idealStart uint64
	if endTime >= depthSec {
		idealStart = endTime - depthSec
	}
	// clamp to the activation timestamp: never backfill before activation.
	startTime := max(idealStart, i.activationTimestamp)
	i.log.Info("log backfill starting",
		"verification_start", firstVerifiable, "activation", i.activationTimestamp,
		"depth", i.logBackfillDepth, "start_time", startTime, "end_time", endTime,
		"chains", len(i.chains))

	// backfill every chain in parallel over [startTime, endTime]
	errCh := make(chan error, len(i.chains))
	wg := sync.WaitGroup{}
	wg.Add(len(i.chains))
	for _, chain := range i.chains {
		go func(chain cc.ChainContainer) {
			defer wg.Done()
			chainStartTime := startTime
			// if we can identify the genesis time, use it to clamp the start time
			// if we can't, we'd either fail now or later when trying to use the value
			if genesisTime, err := chain.BlockNumberToTimestamp(i.ctx, 0); err == nil &&
				genesisTime > startTime {
				chainStartTime = genesisTime
			}
			startNum, err := chain.TimestampToBlockNumber(i.ctx, chainStartTime)
			if err != nil {
				errCh <- fmt.Errorf("chain %s: timestamp to block number for start %d: %w", chain.ID(), chainStartTime, err)
				i.log.Error("log backfill: timestamp to block number for start", "chain", chain.ID(), "err", err)
				return
			}
			endNum, err := chain.TimestampToBlockNumber(i.ctx, endTime)
			if err != nil {
				errCh <- fmt.Errorf("chain %s: timestamp to block number for end %d: %w", chain.ID(), endTime, err)
				i.log.Error("log backfill: timestamp to block number for end", "chain", chain.ID(), "err", err)
				return
			}
			i.log.Info("log backfill started for chain",
				"chain", chain.ID(), "from", startNum, "to", endNum,
				"start_time", chainStartTime, "end_time", endTime,
				"depth", i.logBackfillDepth)
			if err := i.backfillChain(i.ctx, chain.ID(), chain, startNum, endNum); err != nil {
				errCh <- fmt.Errorf("chain %s: backfill: %w", chain.ID(), err)
				return
			}
		}(chain)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		return 0, err
	}
	i.log.Info("log backfill completed",
		"verification_start", firstVerifiable, "start_time", startTime, "end_time", endTime,
		"chains", len(i.chains))
	return endTime, nil
}

func (i *Interop) backfillChain(ctx context.Context, cid eth.ChainID, chain cc.ChainContainer, startNum, endNum uint64) error {
	requestedStart := startNum
	startedAt := time.Now()
	db := i.logsDBs[cid]
	// This is a startup best-effort repair for pre-existing logsDB reorg drift,
	// separate from the normal interop observation/apply loop. It does not close
	// the window where an L2 reorg lands after reconciliation/backfill and before
	// normal interop persists its first frontier block. In that case the write path
	// fails with ErrParentHashMismatch or ErrStaleLogsDB instead of appending
	// inconsistent logs.
	if err := i.reconcileLogsDBTail(ctx, cid, chain, db); err != nil {
		return err
	}
	if latest, has := db.LatestSealedBlock(); has {
		startNum = latest.Number + 1
	}
	if startNum > endNum {
		i.metrics.LogBackfillProgress.WithLabelValues(cid.String()).Set(1)
		i.log.Info("log backfill complete for chain",
			"chain", cid, "from", requestedStart, "to", endNum,
			"sealed", 0, "elapsed", time.Since(startedAt),
			"already_complete", true)
		return nil
	}
	totalBlocks := endNum - startNum + 1
	nextProgressLog := 0.05
	lastProgressLog := startedAt
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

		if totalBlocks > 0 {
			progress := float64(num-startNum+1) / float64(totalBlocks)
			i.metrics.LogBackfillProgress.WithLabelValues(cid.String()).Set(progress)
			if progress >= nextProgressLog || progress >= 1 || time.Since(lastProgressLog) >= 30*time.Second {
				i.log.Info("log backfill progress",
					"chain", cid, "progress", progress,
					"current", num, "from", startNum, "to", endNum,
					"sealed", num-startNum+1, "total", totalBlocks,
					"elapsed", time.Since(startedAt))
				for nextProgressLog <= progress {
					nextProgressLog += 0.05
				}
				lastProgressLog = time.Now()
			}
		}
	}
	i.log.Info("log backfill complete for chain",
		"chain", cid, "from", startNum, "to", endNum,
		"sealed", totalBlocks, "elapsed", time.Since(startedAt),
		"already_complete", false)
	return nil
}

// reconcileLogsDBTail trims tail blocks whose hash no longer matches canonical,
// so backfill resumes from a block that is still in force. Without this, an L2
// reorg that occurs while supernode is offline leaves the tail diverged and the
// first seal on resume loops forever on ErrParentHashMismatch.
func (i *Interop) reconcileLogsDBTail(ctx context.Context, cid eth.ChainID, chain cc.ChainContainer, db LogsDB) error {
	latest, has := db.LatestSealedBlock()
	if !has {
		return nil
	}
	latestOut, err := chain.OutputV0AtBlockNumber(ctx, latest.Number)
	if err != nil {
		return fmt.Errorf("chain %s: output at block %d during logsDB reconcile: %w", cid, latest.Number, err)
	}
	if latestOut.BlockHash == latest.Hash {
		return nil
	}

	first, err := db.FirstSealedBlock()
	if err != nil {
		return fmt.Errorf("chain %s: first sealed block during reconcile: %w", cid, err)
	}

	// Walk back from latest.Number-1 looking for the deepest sealed block whose
	// hash still matches canonical. latest itself is already known to diverge.
	for n := latest.Number; n > first.Number; {
		n--
		seal, err := db.FindSealedBlock(n)
		if err != nil {
			return fmt.Errorf("chain %s: find sealed block %d during reconcile: %w", cid, n, err)
		}
		out, err := chain.OutputV0AtBlockNumber(ctx, n)
		if err != nil {
			return fmt.Errorf("chain %s: output at block %d during reconcile: %w", cid, n, err)
		}
		if seal.Hash != out.BlockHash {
			continue
		}
		i.log.Warn("rewinding logsDB to last canonical block",
			"chain", cid, "rewindTo", n, "trimmedTipNumber", latest.Number,
			"trimmedTipStored", latest.Hash, "trimmedTipCanonical", latestOut.BlockHash)
		if err := db.Rewind(eth.BlockID{Number: n, Hash: seal.Hash}); err != nil {
			return fmt.Errorf("chain %s: rewind logsDB during reconcile: %w", cid, err)
		}
		return nil
	}

	i.log.Warn("entire logsDB diverges from canonical; clearing",
		"chain", cid, "firstSealed", first.Number, "latestSealed", latest.Number)
	if err := db.Clear(); err != nil {
		return fmt.Errorf("chain %s: clear logsDB during reconcile: %w", cid, err)
	}
	return nil
}
