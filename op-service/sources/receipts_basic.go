package sources

import (
	"context"
	"io"
	"sync"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru/v2"
)

type receiptsBatchCall = batching.IterativeBatchCall[common.Hash, *optypes.Receipt]

// maxPendingReceiptsCalls bounds the uncompleted batch calls kept for a retry, so that blocks whose
// receipts fail and are never requested again don't accumulate.
const maxPendingReceiptsCalls = 16

type BasicRPCReceiptsFetcher struct {
	client       rpcClient
	maxBatchSize int

	// calls caches uncompleted batch calls
	calls   *lru.Cache[common.Hash, *receiptsBatchCall]
	callsMu sync.Mutex
}

func NewBasicRPCReceiptsFetcher(client rpcClient, maxBatchSize int) *BasicRPCReceiptsFetcher {
	calls, _ := lru.New[common.Hash, *receiptsBatchCall](maxPendingReceiptsCalls)
	return &BasicRPCReceiptsFetcher{
		client:       client,
		maxBatchSize: maxBatchSize,
		calls:        calls,
	}
}

// FetchReceipts fetches receipts for the given block and transaction hashes
// it does not validate receipts, and expects the caller to do so
func (f *BasicRPCReceiptsFetcher) FetchReceipts(ctx context.Context, blockInfo eth.BlockInfo, txHashes []common.Hash) (optypes.Receipts, error) {
	block := eth.ToBlockID(blockInfo)
	call := f.getOrCreateBatchCall(block.Hash, txHashes)

	// Fetch all receipts
	for {
		if err := call.Fetch(ctx); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}
	res, err := call.Result()
	if err != nil {
		return nil, err
	}
	// call successful, remove from cache
	f.deleteBatchCall(block.Hash)
	return res, nil
}

func (f *BasicRPCReceiptsFetcher) getOrCreateBatchCall(blockHash common.Hash, txHashes []common.Hash) *receiptsBatchCall {
	f.callsMu.Lock()
	defer f.callsMu.Unlock()
	if call, ok := f.calls.Get(blockHash); ok {
		return call
	}
	call := batching.NewIterativeBatchCall[common.Hash, *optypes.Receipt](
		txHashes,
		makeReceiptRequest,
		f.client.BatchCallContext,
		f.client.CallContext,
		f.maxBatchSize,
	)
	f.calls.Add(blockHash, call)
	return call
}

func (f *BasicRPCReceiptsFetcher) deleteBatchCall(blockHash common.Hash) {
	f.callsMu.Lock()
	defer f.callsMu.Unlock()
	f.calls.Remove(blockHash)
}

func makeReceiptRequest(txHash common.Hash) (*optypes.Receipt, rpc.BatchElem) {
	out := new(optypes.Receipt)
	return out, rpc.BatchElem{
		Method: "eth_getTransactionReceipt",
		Args:   []any{txHash},
		Result: &out, // receipt may become nil, double pointer is intentional
	}
}
