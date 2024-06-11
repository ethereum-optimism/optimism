package clsync

import (
	"container/heap"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type payloadAndSize struct {
	envelope *eth.ExecutionPayloadEnvelope
	size     uint64
}

// payloadsByNumber buffers payloads ordered by block number.
// The lowest block number is peeked/popped first.
//
// payloadsByNumber implements heap.Interface: use the heap package methods to modify the queue.
type payloadsByNumber []payloadAndSize

var _ heap.Interface = (*payloadsByNumber)(nil)

func (pq payloadsByNumber) Len() int { return len(pq) }

func (pq payloadsByNumber) Less(i, j int) bool {
	return pq[i].envelope.ExecutionPayload.BlockNumber < pq[j].envelope.ExecutionPayload.BlockNumber
}

// Swap is a heap.Interface method. Do not use this method directly.
func (pq payloadsByNumber) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

// Push is a heap.Interface method. Do not use this method directly, use heap.Push instead.
func (pq *payloadsByNumber) Push(x any) {
	*pq = append(*pq, x.(payloadAndSize))
}

// Pop is a heap.Interface method. Do not use this method directly, use heap.Pop instead.
func (pq *payloadsByNumber) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = payloadAndSize{} // avoid memory leak
	*pq = old[0 : n-1]
	return item
}

const (
	// ~1000 bytes per payload, with some margin for overhead like map data
	payloadMemFixedCost uint64 = 1000
	// 24 bytes per tx overhead (size of slice header in memory)
	payloadTxMemOverhead uint64 = 24
)

func payloadMemSize(p *eth.ExecutionPayloadEnvelope) uint64 {
	out := payloadMemFixedCost
	if p == nil {
		return out
	}
	// 24 byte overhead per tx
	for _, tx := range p.ExecutionPayload.Transactions {
		out += uint64(len(tx)) + payloadTxMemOverhead
	}
	return out
}

// PayloadsQueue buffers payloads by block number.
// PayloadsQueue is not safe to use concurrently.
// PayloadsQueue exposes typed Push/Peek/Pop methods to use the queue,
// without the need to use heap.Push/heap.Pop as caller.
// PayloadsQueue maintains a MaxSize by counting and tracking sizes of added eth.ExecutionPayload entries.
// When the size grows too large, the first (lowest block-number) payload is removed from the queue.
// PayloadsQueue allows entries with same block number, but does not allow duplicate blocks
type PayloadsQueue struct {
	pq          payloadsByNumber
	currentSize uint64
	MaxSize     uint64
	blockHashes map[common.Hash]struct{}
	SizeFn      func(p *eth.ExecutionPayloadEnvelope) uint64
	log         log.Logger
}

func NewPayloadsQueue(log log.Logger, maxSize uint64, sizeFn func(p *eth.ExecutionPayloadEnvelope) uint64) *PayloadsQueue {
	return &PayloadsQueue{
		pq:          nil,
		currentSize: 0,
		MaxSize:     maxSize,
		blockHashes: make(map[common.Hash]struct{}),
		SizeFn:      sizeFn,
		log:         log,
	}
}

func (upq *PayloadsQueue) Len() int {
	return len(upq.pq)
}

func (upq *PayloadsQueue) MemSize() uint64 {
	return upq.currentSize
}

// Push adds the payload to the queue, in O(log(N)).
//
// Don't DoS ourselves by buffering too many unsafe payloads.
// If the queue size after pushing exceed the allowed memory, then pop payloads until memory is not exceeding anymore.
//
// We prefer higher block numbers over lower block numbers, since lower block numbers are more likely to be conflicts and/or read from L1 sooner.
// The higher payload block numbers can be preserved, and once L1 contents meets these, they can all be processed in order.
func (upq *PayloadsQueue) Push(e *eth.ExecutionPayloadEnvelope) error {
	if e == nil || e.ExecutionPayload == nil {
		return errors.New("cannot add nil payload")
	}
	if _, ok := upq.blockHashes[e.ExecutionPayload.BlockHash]; ok {
		return fmt.Errorf("cannot add duplicate payload %s", e.ExecutionPayload.ID())
	}
	size := upq.SizeFn(e)
	if size > upq.MaxSize {
		return fmt.Errorf("cannot add payload %s, payload mem size %d is larger than max queue size %d", e.ExecutionPayload.ID(), size, upq.MaxSize)
	}
	heap.Push(&upq.pq, payloadAndSize{
		envelope: e,
		size:     size,
	})
	upq.currentSize += size
	for upq.currentSize > upq.MaxSize {
		env := upq.Pop()
		upq.log.Info("Dropping payload from payload queue because the payload queue is too large", "id", env.ExecutionPayload.ID())
	}
	upq.blockHashes[e.ExecutionPayload.BlockHash] = struct{}{}
	return nil
}

// Peek retrieves the payload with the lowest block number from the queue in O(1), or nil if the queue is empty.
func (upq *PayloadsQueue) Peek() *eth.ExecutionPayloadEnvelope {
	if len(upq.pq) == 0 {
		return nil
	}
	// peek into the priority queue, the first element is the highest priority (lowest block number).
	// This does not apply to other elements, those are structured like a heap.
	return upq.pq[0].envelope
}

// Pop removes the payload with the lowest block number from the queue in O(log(N)),
// and may return nil if the queue is empty.
func (upq *PayloadsQueue) Pop() *eth.ExecutionPayloadEnvelope {
	if len(upq.pq) == 0 {
		return nil
	}
	ps := heap.Pop(&upq.pq).(payloadAndSize) // nosemgrep
	upq.currentSize -= ps.size
	// remove the key from the block hashes map
	delete(upq.blockHashes, ps.envelope.ExecutionPayload.BlockHash)
	return ps.envelope
}
