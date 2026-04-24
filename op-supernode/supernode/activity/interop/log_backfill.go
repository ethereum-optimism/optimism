package interop

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

func (i *Interop) runLogBackfill() (uint64, error) {
	if i.logBackfillDepth <= 0 {
		return 0, nil
	}
	if len(i.chains) == 0 {
		return 0, nil
	}

	// identify the minimum cross-safe time across all chains
	minCrossSafeTime := uint64(math.MaxUint64)
	for _, chain := range i.chains {
		syncStatus, err := chain.SyncStatus(i.ctx)
		if err != nil {
			return 0, fmt.Errorf("chain %s: sync status: %w", chain.ID(), err)
		}
		// watch that local safe advances to become non-zero
		// local safe should advance even if cross safe has nothing to work from
		if syncStatus.LocalSafeL2.Number == 0 {
			return 0, fmt.Errorf("chain %s: local safe L2 number is 0", chain.ID())
		}
		i.log.Debug("log backfill: sync status",
			"chain", chain.ID(), "safe", syncStatus.SafeL2, "localSafe", syncStatus.LocalSafeL2)
		minCrossSafeTime = min(minCrossSafeTime, syncStatus.SafeL2.Time)
	}

	// if activation falls after the backfill range end, don't backfill
	if i.activationTimestamp > minCrossSafeTime {
		i.log.Info("log backfill: activation timestamp falls after backfill range end, skipping backfill",
			"activationTimestamp", i.activationTimestamp, "minCrossSafeTime", minCrossSafeTime)
		return 0, nil
	}

	// naively, end minus depth is the ideal backfill start.
	// guard the subtraction so a young chain (crossSafe < depth) doesn't wrap.
	depthSec := uint64(i.logBackfillDepth.Seconds())
	var idealStart uint64
	if minCrossSafeTime >= depthSec {
		idealStart = minCrossSafeTime - depthSec
	}
	// clamp to the activation timestamp: never backfill before activation.
	startTime := max(idealStart, i.activationTimestamp)

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
				errCh <- fmt.Errorf("chain %s: timestamp to block number for start %d: %w", chain.ID(), startTime, err)
				i.log.Error("log backfill: timestamp to block number for start", "chain", chain.ID(), "err", err)
				return
			}
			endNum, err := chain.TimestampToBlockNumber(i.ctx, minCrossSafeTime)
			if err != nil {
				errCh <- fmt.Errorf("chain %s: timestamp to block number for end %d: %w", chain.ID(), minCrossSafeTime, err)
				i.log.Error("log backfill: timestamp to block number for end", "chain", chain.ID(), "err", err)
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
		return 0, err
	}
	return minCrossSafeTime, nil
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
