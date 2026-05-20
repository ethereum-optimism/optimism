package interop

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// advanceColdStartInit runs one best-effort pass at cold-start initialization:
// it collects every chain's first SafeDB entry timestamp, picks
// verificationStartTimestamp = max(activation, max_c T_c), runs backfill, and
// signals advance=true on success. Returns advance=false when any chain's
// SafeDB is still empty (caller backs off and retries). Errors from the
// backfill phase are fatal.
func (i *Interop) advanceColdStartInit() (bool, error) {
	i.backfillAttempts.Add(1)

	perChainTS, ready, err := i.collectFirstSafeHeadTimestamps()
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}

	verificationStart := i.activationTimestamp
	for _, ts := range perChainTS {
		if ts > verificationStart {
			verificationStart = ts
		}
	}
	i.verificationStartTimestamp = verificationStart
	// Flip initialized before backfill: backfill seals into logsDB, and
	// sealBlockDataIntoLogsDB queries firstVerifiableTimestamp to validate
	// the timestamp gap. That accessor returns ErrNotStarted while
	// initialized is false.
	i.initialized.Store(true)

	if err := i.runColdStartBackfill(verificationStart); err != nil {
		return false, fmt.Errorf("backfill: %w", err)
	}
	i.backfillCompleted.Store(true)
	return true, nil
}

// collectFirstSafeHeadTimestamps queries every chain's SafeDB for its first
// entry timestamp in parallel. Returns ready=false (without error) if any
// chain has no entries yet; the caller backs off and retries. Other errors
// are reported as-is.
func (i *Interop) collectFirstSafeHeadTimestamps() (map[eth.ChainID]uint64, bool, error) {
	type res struct {
		id  eth.ChainID
		ts  uint64
		err error
	}
	results := make(chan res, len(i.chains))
	for _, chain := range i.chains {
		go func(c cc.ChainContainer) {
			ts, err := c.FirstSafeHeadTimestamp(i.ctx)
			results <- res{id: c.ID(), ts: ts, err: err}
		}(chain)
	}
	out := make(map[eth.ChainID]uint64, len(i.chains))
	var firstErr error
	emptyAny := false
	for range i.chains {
		r := <-results
		if r.err != nil {
			if errors.Is(r.err, cc.ErrSafeDBNotReady) {
				emptyAny = true
				i.log.Debug("interop cold start: chain SafeDB empty, waiting", "chain", r.id)
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("chain %s: first safe head timestamp: %w", r.id, r.err)
			}
			continue
		}
		out[r.id] = r.ts
	}
	if firstErr != nil {
		return nil, false, firstErr
	}
	if emptyAny {
		return nil, false, nil
	}
	return out, true, nil
}

// runColdStartBackfill seals logs over the configured backfill window leading
// up to verificationStart. The per-chain lower bound is
// max(activationTimestamp, perChainGenesisTime, verificationStart - depth).
// Returns nil if logBackfillDepth is zero or no chains are configured.
func (i *Interop) runColdStartBackfill(verificationStart uint64) error {
	if i.logBackfillDepth <= 0 {
		return nil
	}
	if len(i.chains) == 0 {
		return nil
	}
	if verificationStart == 0 {
		return fmt.Errorf("invalid verificationStartTimestamp 0 for backfill")
	}
	endTime := verificationStart - 1

	depthSec := uint64(i.logBackfillDepth.Seconds())
	var depthFloor uint64
	if endTime >= depthSec {
		depthFloor = endTime - depthSec
	}
	commonStart := max(depthFloor, i.activationTimestamp)
	if commonStart > endTime {
		return nil
	}

	errCh := make(chan error, len(i.chains))
	wg := sync.WaitGroup{}
	wg.Add(len(i.chains))
	for _, chain := range i.chains {
		go func(chain cc.ChainContainer) {
			defer wg.Done()
			chainStart := commonStart
			genesisTime, err := chain.BlockNumberToTimestamp(i.ctx, 0)
			if err != nil {
				errCh <- fmt.Errorf("chain %s: genesis timestamp: %w", chain.ID(), err)
				return
			}
			if genesisTime > chainStart {
				chainStart = genesisTime
			}
			if chainStart > endTime {
				return
			}
			startNum, err := chain.TimestampToBlockNumber(i.ctx, chainStart)
			if err != nil {
				errCh <- fmt.Errorf("chain %s: timestamp to block number for start %d: %w", chain.ID(), chainStart, err)
				return
			}
			endNum, err := chain.TimestampToBlockNumber(i.ctx, endTime)
			if err != nil {
				errCh <- fmt.Errorf("chain %s: timestamp to block number for end %d: %w", chain.ID(), endTime, err)
				return
			}
			i.log.Info("log backfill: sealing logs",
				"chain", chain.ID(), "from", startNum, "to", endNum)
			if err := i.backfillChain(i.ctx, chain.ID(), chain, startNum, endNum); err != nil {
				errCh <- fmt.Errorf("chain %s: backfill: %w", chain.ID(), err)
				return
			}
		}(chain)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		return err
	}
	return nil
}

// backfillChain seals every canonical block in [startNum, endNum] into the
// chain's logsDB. Calls reconcileLogsDBTail first to drop any tail that
// diverged from canonical while the supernode was offline (only meaningful
// during cold-start backfill: a verifiedDB-resume path never enters here).
func (i *Interop) backfillChain(ctx context.Context, cid eth.ChainID, chain cc.ChainContainer, startNum, endNum uint64) error {
	db := i.logsDBs[cid]
	if err := i.reconcileLogsDBTail(ctx, cid, chain, db); err != nil {
		return err
	}
	if latest, has := db.LatestSealedBlock(); has {
		startNum = latest.Number + 1
	}
	if startNum > endNum {
		return nil
	}
	totalBlocks := endNum - startNum + 1
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
		}
	}
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
