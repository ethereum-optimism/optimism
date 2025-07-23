package queue

import (
	"github.com/ethereum-optimism/optimism/op-service/queue"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockWithSizeEstimate struct {
	*types.Block
	daSize  uint64
	rawSize uint64
}

func NewBlockWithEstimatedSize(inner *types.Block) *BlockWithSizeEstimate {
	return &BlockWithSizeEstimate{
		Block: inner,
	}
}

func (b *BlockWithSizeEstimate) Estimate() (uint64, uint64) {
	if b.daSize == 0 {
		b.daSize, b.rawSize = estimateBatchSize(b.Block)
	}
	return b.daSize, b.rawSize
}

// estimateBatchSize returns the estimated size of the block in a batch both with compression ('daSize') and without
// ('rawSize').
func estimateBatchSize(block *types.Block) (daSize, rawSize uint64) {
	daSize = uint64(70) // estimated overhead of batch metadata
	rawSize = uint64(70)
	for _, tx := range block.Transactions() {
		// Deposit transactions are not included in batches
		if tx.IsDepositTx() {
			continue
		}
		bigSize := tx.RollupCostData().EstimatedDASize()
		if bigSize.IsUint64() { // this should always be true, but if not just ignore
			daSize += bigSize.Uint64()
		}
		// Add 2 for the overhead of encoding the tx bytes in a RLP list
		rawSize += tx.Size() + 2
	}
	return
}

type Queue struct {
	q           queue.Queue[*BlockWithSizeEstimate]
	pendingSize uint64
}

func New(blocks ...*BlockWithSizeEstimate) *Queue {
	q := &Queue{
		q: queue.Queue[*BlockWithSizeEstimate]{},
	}
	for _, block := range blocks {
		q.Enqueue(block)
	}
	return q
}

func (q *Queue) PendingSize() uint64 {
	return q.pendingSize
}

func (q *Queue) Clear() {
	q.q.Clear()
}

func (q Queue) Len() int {
	return q.q.Len()
}

func (q *Queue) Peek() (*BlockWithSizeEstimate, bool) {
	return q.q.Peek()
}

func (q *Queue) PeekN(i int) (*BlockWithSizeEstimate, bool) {
	return q.q.PeekN(i)
}

func (q *Queue) DequeueN(n int) ([]*BlockWithSizeEstimate, bool) {
	blocks, ok := q.q.DequeueN(n)
	if !ok {
		return nil, ok
	}
	for _, block := range blocks {
		daSize, _ := block.Estimate()
		q.pendingSize -= daSize
	}
	return blocks, ok
}

func (q *Queue) Enqueue(blocks ...*BlockWithSizeEstimate) {
	q.q.Enqueue(blocks...)
	for _, block := range blocks {
		daSize, _ := block.Estimate()
		q.pendingSize += daSize
	}
}
