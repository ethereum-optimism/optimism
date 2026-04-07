package interop

import (
	"context"
	"fmt"
	"sort"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// chainBackfillProgress tracks per-chain sequential log sealing for --interop.log-backfill-depth.
type chainBackfillProgress struct {
	done      bool
	nextBlock uint64 // next L2 block number to seal; 0 = not yet computed for this chain
}

func sortedChainIDs(chains map[eth.ChainID]cc.ChainContainer) []eth.ChainID {
	out := make([]eth.ChainID, 0, len(chains))
	for id := range chains {
		out = append(out, id)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Cmp(out[b]) < 0 })
	return out
}

// runLogBackfillStep seals at most one L2 block across all chains (round-robin by sorted ID).
// Returns (true, nil) if a block was sealed; (false, nil) if backfill is finished or nothing to do.
func (i *Interop) runLogBackfillStep() (bool, error) {
	if i.logBackfillDepth <= 0 {
		return false, nil
	}

	i.backfillMu.Lock()
	if i.logBackfillComplete {
		i.backfillMu.Unlock()
		return false, nil
	}
	if i.backfillProgress == nil {
		i.backfillProgress = make(map[eth.ChainID]*chainBackfillProgress)
	}
	i.backfillMu.Unlock()

	ctx := i.ctx
	for _, cid := range sortedChainIDs(i.chains) {
		sealed, err := i.trySealNextBackfillBlock(ctx, cid)
		if err != nil {
			return false, err
		}
		if sealed {
			return true, nil
		}
	}

	i.backfillMu.Lock()
	allDone := true
	for _, cid := range sortedChainIDs(i.chains) {
		p := i.backfillProgress[cid]
		if p == nil || !p.done {
			allDone = false
			break
		}
	}
	i.backfillMu.Unlock()
	if !allDone {
		i.log.Warn("log backfill inconsistent state")
		return false, nil
	}

	ceiling := make(map[eth.ChainID]uint64, len(i.chains))
	for _, cid := range sortedChainIDs(i.chains) {
		ss, err := i.chains[cid].SyncStatus(ctx)
		if err != nil {
			return false, fmt.Errorf("chain %s: sync status for backfill ceiling: %w", cid, err)
		}
		ceiling[cid] = ss.LocalSafeL2.Number
	}

	i.backfillMu.Lock()
	i.logBackfillComplete = true
	i.postBackfillLocalSafeCeiling = ceiling
	i.backfillMu.Unlock()
	i.log.Info("interop log backfill complete")
	return false, nil
}

func (i *Interop) trySealNextBackfillBlock(ctx context.Context, cid eth.ChainID) (bool, error) {
	chain := i.chains[cid]

	i.backfillMu.Lock()
	p := i.backfillProgress[cid]
	if p != nil && p.done {
		i.backfillMu.Unlock()
		return false, nil
	}
	i.backfillMu.Unlock()

	ss, err := chain.SyncStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("chain %s: sync status: %w", cid, err)
	}
	tipTs := ss.UnsafeL2.Time
	safeNum := ss.LocalSafeL2.Number

	Tlo := LogBackfillLowerBound(tipTs, i.activationTimestamp, i.logBackfillDepth)
	startNum, err := chain.TimestampToBlockNumber(ctx, Tlo)
	if err != nil {
		startNum = 0
	}
	if startNum > safeNum {
		i.backfillMu.Lock()
		if i.backfillProgress[cid] == nil {
			i.backfillProgress[cid] = &chainBackfillProgress{done: true}
		} else {
			i.backfillProgress[cid].done = true
		}
		i.backfillMu.Unlock()
		return false, nil
	}

	i.backfillMu.Lock()
	p = i.backfillProgress[cid]
	if p == nil {
		p = &chainBackfillProgress{nextBlock: startNum}
		i.backfillProgress[cid] = p
	}
	if p.done {
		i.backfillMu.Unlock()
		return false, nil
	}
	next := p.nextBlock
	if next == 0 {
		next = startNum
		p.nextBlock = next
	}
	if next > safeNum {
		p.done = true
		i.backfillMu.Unlock()
		return false, nil
	}
	i.backfillMu.Unlock()

	out, err := chain.OutputV0AtBlockNumber(ctx, next)
	if err != nil {
		return false, fmt.Errorf("chain %s: output at block %d: %w", cid, next, err)
	}
	bid := eth.BlockID{Hash: out.BlockHash, Number: next}
	blockInfo, receipts, err := chain.FetchReceipts(ctx, bid)
	if err != nil {
		return false, fmt.Errorf("chain %s: fetch receipts %d: %w", cid, next, err)
	}

	i.inLogBackfill.Store(true)
	err = i.sealBlockDataIntoLogsDB(cid, bid, blockInfo, receipts, blockInfo.Time())
	i.inLogBackfill.Store(false)
	if err != nil {
		return false, err
	}

	i.backfillMu.Lock()
	p = i.backfillProgress[cid]
	p.nextBlock = next + 1
	if p.nextBlock > safeNum {
		p.done = true
	}
	i.backfillMu.Unlock()
	return true, nil
}

func (i *Interop) shouldSkipValidationUsingBackfillCeiling(blocks map[eth.ChainID]eth.BlockID) bool {
	i.backfillMu.Lock()
	ceiling := i.postBackfillLocalSafeCeiling
	i.backfillMu.Unlock()
	if ceiling == nil || len(blocks) == 0 {
		return false
	}
	for cid, bid := range blocks {
		lim, ok := ceiling[cid]
		if !ok || bid.Number > lim {
			return false
		}
	}
	return true
}

func maxL1Inclusion(heads map[eth.ChainID]eth.BlockID) eth.BlockID {
	var max eth.BlockID
	first := true
	for _, h := range heads {
		if first || h.Number > max.Number {
			max = h
			first = false
		}
	}
	return max
}

func cloneBlockIDMap(m map[eth.ChainID]eth.BlockID) map[eth.ChainID]eth.BlockID {
	if m == nil {
		return nil
	}
	out := make(map[eth.ChainID]eth.BlockID, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
