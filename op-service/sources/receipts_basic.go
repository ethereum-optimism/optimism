package sources

import (
	"context"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type receiptsBatchCall = batching.IterativeBatchCall[common.Hash, *types.Receipt]

type BasicRPCReceiptsFetcher struct {
	client       rpcClient
	maxBatchSize int

	// calls caches uncompleted batch calls
	calls   map[common.Hash]*receiptsBatchCall
	callsMu sync.Mutex
}

func NewBasicRPCReceiptsFetcher(client rpcClient, maxBatchSize int) *BasicRPCReceiptsFetcher {
	return &BasicRPCReceiptsFetcher{
		client:       client,
		maxBatchSize: maxBatchSize,
		calls:        make(map[common.Hash]*receiptsBatchCall),
	}
}

func (f *BasicRPCReceiptsFetcher) FetchReceipts(ctx context.Context, block eth.BlockID, txHashes []common.Hash) (types.Receipts, error) {
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
	if call, ok := f.calls[blockHash]; ok {
		return call
	}
	call := batching.NewIterativeBatchCall[common.Hash, *types.Receipt](
		txHashes,
		makeReceiptRequest,
		f.client.BatchCallContext,
		f.client.CallContext,
		f.maxBatchSize,
	)
	f.calls[blockHash] = call
	return call
}

func (f *BasicRPCReceiptsFetcher) deleteBatchCall(blockHash common.Hash) {
	f.callsMu.Lock()
	defer f.callsMu.Unlock()
	delete(f.calls, blockHash)
}

func makeReceiptRequest(txHash common.Hash) (*types.Receipt, rpc.BatchElem) {
	out := new(types.Receipt)
	return out, rpc.BatchElem{
		Method: "eth_getTransactionReceipt",
		Args:   []any{txHash},
		Result: &out, // receipt may become nil, double pointer is intentional
	}
}
