package queue

import (
	"github.com/ethereum-optimism/optimism/op-service/queue"
	"github.com/ethereum/go-ethereum/core/types"
)

type Block struct {
	*types.Block
	daBytes uint64
}

func NewBlock(inner *types.Block) *Block {
	return &Block{
		Block: inner,
	}
}

func (b *Block) DASizeEstimate() uint64 {
	if b.daBytes == 0 {
		b.daBytes, _ = estimateBatchSize(b.Block)
	}
	return b.daBytes
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
	q              queue.Queue[*Block]
	daSizeEstimate uint64
}

func New(blocks ...*Block) *Queue {
	q := &Queue{
		q: queue.Queue[*Block]{},
	}
	for _, block := range blocks {
		q.Enqueue(block)
	}
	return q
}

func (q *Queue) DASizeEstimate() uint64 {
	return q.daSizeEstimate
}

func (q *Queue) Clear() {
	q.q.Clear()
}

func (q *Queue) Len() int {
	return q.q.Len()
}

func (q *Queue) MustPeek() *Block {
	return q.q[0]
}

func (q *Queue) MustPeekN(i int) *Block {
	return q.q[i]
}

func (q *Queue) Peek() (*Block, bool) {
	return q.q.Peek()
}

func (q *Queue) PeekN(i int) (*Block, bool) {
	return q.q.PeekN(i)
}

func (q *Queue) DequeueN(n int) ([]*Block, bool) {
	blocks, ok := q.q.DequeueN(n)
	if !ok {
		return nil, ok
	}
	for _, block := range blocks {
		q.daSizeEstimate -= block.DASizeEstimate()
	}
	return blocks, ok
}

func (q *Queue) Enqueue(blocks ...*Block) {
	q.q.Enqueue(blocks...)
	for _, block := range blocks {
		q.daSizeEstimate += block.DASizeEstimate()
	}
}
