package filter

import (
	"context"

	gethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type blockFetch struct {
	blockNum  uint64
	blockInfo eth.BlockInfo
	receipts  gethTypes.Receipts
	err       error
}

// BlockPrefetcher fetches block data concurrently and returns results in block-number order.
type BlockPrefetcher interface {
	FetchRange(ctx context.Context, startBlock, endBlock uint64) <-chan blockFetch
}

type blockPrefetcher struct {
	ethClient   EthClient
	concurrency int
}

// NewBlockPrefetcher creates a bounded concurrent prefetcher backed by the given eth client.
func NewBlockPrefetcher(ethClient EthClient, concurrency int) BlockPrefetcher {
	return &blockPrefetcher{
		ethClient:   ethClient,
		concurrency: concurrency,
	}
}

func (p *blockPrefetcher) FetchRange(ctx context.Context, startBlock, endBlock uint64) <-chan blockFetch {
	if startBlock > endBlock || p.concurrency <= 0 {
		results := make(chan blockFetch)
		close(results)
		return results
	}

	results := make(chan blockFetch, p.concurrency)

	go func() {
		defer close(results)

		totalBlocks := endBlock - startBlock + 1
		window := uint64(p.concurrency)
		if window > totalBlocks {
			window = totalBlocks
		}

		slots := make(map[uint64]chan blockFetch, window)
		startFetch := func(blockNum uint64) {
			slot := make(chan blockFetch, 1)
			slots[blockNum] = slot
			go func() {
				fetched := p.fetchBlock(ctx, blockNum)
				select {
				case slot <- fetched:
				case <-ctx.Done():
				}
			}()
		}

		nextFetch := startBlock
		for i := uint64(0); i < window; i++ {
			startFetch(nextFetch)
			nextFetch++
		}

		for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
			var fetched blockFetch
			select {
			case fetched = <-slots[blockNum]:
			case <-ctx.Done():
				return
			}
			delete(slots, blockNum)

			if nextFetch <= endBlock {
				startFetch(nextFetch)
				nextFetch++
			}

			select {
			case results <- fetched:
			case <-ctx.Done():
				return
			}
		}
	}()

	return results
}

func (p *blockPrefetcher) fetchBlock(ctx context.Context, blockNum uint64) blockFetch {
	blockInfo, receipts, err := p.ethClient.FetchReceiptsByNumber(ctx, blockNum)
	return blockFetch{
		blockNum:  blockNum,
		blockInfo: blockInfo,
		receipts:  receipts,
		err:       err,
	}
}
