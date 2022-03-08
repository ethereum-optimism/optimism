package l1

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const MaxConcurrentFetchesPerCall = 10
const MaxReceiptRetry = 3

type EthClient interface {
	BlockByHash(context.Context, common.Hash) (*types.Block, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}

type Downloader struct {
	client EthClient
	// log    log.Logger
}

func NewDownloader(client EthClient) *Downloader {
	return &Downloader{client: client}
}

func (dl Downloader) Fetch(ctx context.Context, id eth.BlockID) (*types.Block, []*types.Receipt, error) {
	block, err := dl.client.BlockByHash(ctx, id.Hash)
	if err != nil {
		return nil, nil, err
	}
	txs := block.Transactions()
	receipts := make([]*types.Receipt, len(txs))

	semaphoreChan := make(chan struct{}, MaxConcurrentFetchesPerCall)
	defer close(semaphoreChan)
	var retErr error
	var errMu sync.Mutex
	var wg sync.WaitGroup
	for idx, tx := range txs {
		wg.Add(1)
		i := idx
		hash := tx.Hash()
		go func() {
			semaphoreChan <- struct{}{}
			for j := 0; j < MaxReceiptRetry; j++ {
				receipt, err := dl.client.TransactionReceipt(ctx, hash)
				if err != nil && j == MaxReceiptRetry-1 {
					// dl.log.Error("Got error in final retry of fetch", "err", err)
					errMu.Lock()
					retErr = err
					errMu.Unlock()
				} else if err == nil {
					receipts[i] = receipt
					break
				} else {
					time.Sleep(20 * time.Millisecond)
				}
			}
			wg.Done()
			<-semaphoreChan
		}()
	}
	wg.Wait()
	if retErr != nil {
		return nil, nil, retErr
	}
	return block, receipts, nil
}
