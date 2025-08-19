package clsync

import (
	"container/heap"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestPayloadsByNumber(t *testing.T) {
	p := payloadsByNumber{}
	mk := func(i uint64) payloadAndSize {
		return payloadAndSize{
			envelope: &eth.ExecutionPayloadEnvelope{
				ExecutionPayload: &eth.ExecutionPayload{
					BlockNumber: eth.Uint64Quantity(i),
				},
			},
		}
	}
	// add payload A, check it was added
	a := mk(123)
	heap.Push(&p, a)
	require.Equal(t, p.Len(), 1)
	require.Equal(t, p[0], a)

	// add payload B, check it was added in top-priority spot
	b := mk(100)
	heap.Push(&p, b)
	require.Equal(t, p.Len(), 2)
	require.Equal(t, p[0], b)

	// add payload C, check it did not get first like B, since block num is higher
	c := mk(150)
	heap.Push(&p, c)
	require.Equal(t, p.Len(), 3)
	require.Equal(t, p[0], b) // still b

	// pop b
	heap.Pop(&p)
	require.Equal(t, p.Len(), 2)
	require.Equal(t, p[0], a)

	// pop a
	heap.Pop(&p)
	require.Equal(t, p.Len(), 1)
	require.Equal(t, p[0], c)

	// pop c
	heap.Pop(&p)
	require.Equal(t, p.Len(), 0)

	// duplicate entry
	heap.Push(&p, b)
	require.Equal(t, p.Len(), 1)
	heap.Push(&p, b)
	require.Equal(t, p.Len(), 2)
	heap.Pop(&p)
	require.Equal(t, p.Len(), 1)
}

func TestPayloadMemSize(t *testing.T) {
	require.Equal(t, payloadMemFixedCost, payloadMemSize(nil), "nil is same fixed cost")
	require.Equal(t, payloadMemFixedCost, payloadMemSize(&eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{}}), "empty payload fixed cost")
	require.Equal(t, payloadMemFixedCost+payloadTxMemOverhead, payloadMemSize(&eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{Transactions: []eth.Data{nil}}}), "nil tx counts")
	require.Equal(t, payloadMemFixedCost+payloadTxMemOverhead, payloadMemSize(&eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{Transactions: []eth.Data{make([]byte, 0)}}}), "empty tx counts")
	require.Equal(t, payloadMemFixedCost+4*payloadTxMemOverhead+42+1337+0+1,
		payloadMemSize(&eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{Transactions: []eth.Data{
			make([]byte, 42),
			make([]byte, 1337),
			make([]byte, 0),
			make([]byte, 1),
		}}}), "mixed txs")
}

func envelope(payload *eth.ExecutionPayload) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{ExecutionPayload: payload}
}

func TestPayloadsQueue(t *testing.T) {
	pq := NewPayloadsQueue(testlog.Logger(t, log.LvlInfo), payloadMemFixedCost*3, payloadMemSize)
	require.Equal(t, 0, pq.Len())
	require.Nil(t, pq.Peek())
	require.Nil(t, pq.Pop())

	a := envelope(&eth.ExecutionPayload{BlockNumber: 3, BlockHash: common.Hash{3}})
	b := envelope(&eth.ExecutionPayload{BlockNumber: 4, BlockHash: common.Hash{4}})
	c := envelope(&eth.ExecutionPayload{BlockNumber: 5, BlockHash: common.Hash{5}})
	d := envelope(&eth.ExecutionPayload{BlockNumber: 6, BlockHash: common.Hash{6}})
	bAlt := envelope(&eth.ExecutionPayload{BlockNumber: 4, BlockHash: common.Hash{0xff}})
	bDup := envelope(&eth.ExecutionPayload{BlockNumber: 4, BlockHash: common.Hash{4}})

	require.NoError(t, pq.Push(b))
	require.Equal(t, pq.Len(), 1)
	require.Equal(t, pq.Peek(), b)

	require.Error(t, pq.Push(nil), "cannot add nil payloads")

	require.NoError(t, pq.Push(c))
	require.Equal(t, pq.Len(), 2)
	require.Equal(t, pq.MemSize(), 2*payloadMemFixedCost)
	require.Equal(t, pq.Peek(), b, "expecting b to still be the lowest number payload")

	require.NoError(t, pq.Push(a))
	require.Equal(t, pq.Len(), 3)
	require.Equal(t, pq.MemSize(), 3*payloadMemFixedCost)
	require.Equal(t, pq.Peek(), a, "expecting a to be new lowest number")

	require.Equal(t, pq.Pop(), a)
	require.Equal(t, pq.Len(), 2, "expecting to pop the lowest")

	require.Equal(t, pq.Peek(), b, "expecting b to be lowest, compared to c")

	require.Equal(t, pq.Pop(), b)
	require.Equal(t, pq.Len(), 1)
	require.Equal(t, pq.MemSize(), payloadMemFixedCost)

	require.Equal(t, pq.Pop(), c)
	require.Equal(t, pq.Len(), 0, "expecting no items to remain")

	e := envelope(&eth.ExecutionPayload{BlockNumber: 5, Transactions: []eth.Data{make([]byte, payloadMemFixedCost*3+1)}})
	require.Error(t, pq.Push(e), "cannot add payloads that are too large")

	require.NoError(t, pq.Push(b))
	require.Equal(t, pq.Len(), 1, "expecting b")
	require.Equal(t, pq.Peek(), b)
	require.NoError(t, pq.Push(c))
	require.Equal(t, pq.Len(), 2, "expecting b, c")
	require.Equal(t, pq.Peek(), b)
	require.NoError(t, pq.Push(a))
	require.Equal(t, pq.Len(), 3, "expecting a, b, c")
	require.Equal(t, pq.Peek(), a)

	// No duplicates allowed
	require.Error(t, pq.Push(bDup))
	// But reorg data allowed
	require.NoError(t, pq.Push(bAlt))

	require.NoError(t, pq.Push(d))
	require.Equal(t, pq.Len(), 3)
	require.Equal(t, pq.Peek(), b, "expecting b, c, d")
	require.NotContainsf(t, pq.pq[:], a, "a should be dropped after 3 items already exist under max size constraint")
}

func TestPayloadsQueue_ReaddAfterPopAllowed(t *testing.T) {
	pq := NewPayloadsQueue(testlog.Logger(t, log.LvlInfo), payloadMemFixedCost*10, payloadMemSize)
	b := envelope(&eth.ExecutionPayload{BlockNumber: 4, BlockHash: common.Hash{4}})
	require.NoError(t, pq.Push(b))
	require.Equal(t, b, pq.Pop())
	// re-add same hash after pop should be allowed
	require.NoError(t, pq.Push(b))
}

func TestDropInapplicable_PopsMultipleInapplicable(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	pq := NewPayloadsQueue(logger, payloadMemFixedCost*10, payloadMemSize)

	// queue: processed (=unsafe head), old<=safe, old<=unsafe, then applicable next
	processed := envelope(&eth.ExecutionPayload{BlockNumber: 10, BlockHash: common.Hash{0x10}})
	oldSafe := envelope(&eth.ExecutionPayload{BlockNumber: 8, BlockHash: common.Hash{0x08}})
	oldUnsafe := envelope(&eth.ExecutionPayload{BlockNumber: 9, BlockHash: common.Hash{0x09}})
	next := envelope(&eth.ExecutionPayload{BlockNumber: 11, ParentHash: common.Hash{0x10}, BlockHash: common.Hash{0x11}})

	require.NoError(t, pq.Push(processed))
	require.NoError(t, pq.Push(oldUnsafe))
	require.NoError(t, pq.Push(oldSafe))
	require.NoError(t, pq.Push(next))

	ev := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    eth.L2BlockRef{Hash: common.Hash{0x10}, Number: 10},
		SafeL2Head:      eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 9},
		FinalizedL2Head: eth.L2BlockRef{},
	}

	pq.DropInapplicableUnsafePayloads(ev)
	require.Equal(t, 1, pq.Len())
	require.Equal(t, next, pq.Peek())
}

func mkRef(number uint64, hash common.Hash) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:   hash,
		Number: number,
	}
}

func TestDropInapplicable_RemovesAlreadyProcessed(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	pq := NewPayloadsQueue(logger, payloadMemFixedCost*10, payloadMemSize)

	headHash := common.Hash{0xaa}
	headNum := uint64(10)
	processed := envelope(&eth.ExecutionPayload{BlockNumber: eth.Uint64Quantity(headNum), BlockHash: headHash})
	require.NoError(t, pq.Push(processed))

	ev := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    mkRef(headNum, headHash),
		SafeL2Head:      mkRef(0, common.Hash{}),
		FinalizedL2Head: mkRef(0, common.Hash{}),
	}

	pq.DropInapplicableUnsafePayloads(ev)
	require.Equal(t, 0, pq.Len())
}

func TestDropInapplicable_DropOlderThanSafe(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	pq := NewPayloadsQueue(logger, payloadMemFixedCost*10, payloadMemSize)

	payload := envelope(&eth.ExecutionPayload{BlockNumber: eth.Uint64Quantity(8), BlockHash: common.Hash{0x01}})
	require.NoError(t, pq.Push(payload))

	ev := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    mkRef(10, common.Hash{0xaa}),
		SafeL2Head:      mkRef(8, common.Hash{0xbb}),
		FinalizedL2Head: mkRef(0, common.Hash{}),
	}

	pq.DropInapplicableUnsafePayloads(ev)
	require.Equal(t, 0, pq.Len())
}

func TestDropInapplicable_DropOlderThanUnsafe(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	pq := NewPayloadsQueue(logger, payloadMemFixedCost*10, payloadMemSize)

	// Block is newer than safe head but not newer than unsafe head
	payload := envelope(&eth.ExecutionPayload{BlockNumber: eth.Uint64Quantity(10), BlockHash: common.Hash{0x02}})
	require.NoError(t, pq.Push(payload))

	ev := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    mkRef(10, common.Hash{0xaa}),
		SafeL2Head:      mkRef(9, common.Hash{0xbb}),
		FinalizedL2Head: mkRef(0, common.Hash{}),
	}

	pq.DropInapplicableUnsafePayloads(ev)
	require.Equal(t, 0, pq.Len())
}

func TestDropInapplicable_DropNextHeightMismatch(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	pq := NewPayloadsQueue(logger, payloadMemFixedCost*10, payloadMemSize)

	headHash := common.Hash{0xaa}
	// Next height but wrong parent
	payload := envelope(&eth.ExecutionPayload{BlockNumber: eth.Uint64Quantity(11), BlockHash: common.Hash{0x03}, ParentHash: common.Hash{0xff}})
	require.NoError(t, pq.Push(payload))

	ev := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    mkRef(10, headHash),
		SafeL2Head:      mkRef(9, common.Hash{0xbb}),
		FinalizedL2Head: mkRef(0, common.Hash{}),
	}

	pq.DropInapplicableUnsafePayloads(ev)
	require.Equal(t, 0, pq.Len())
}

func TestDropInapplicable_NonAdjacentMismatchReturns(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	pq := NewPayloadsQueue(logger, payloadMemFixedCost*10, payloadMemSize)

	headHash := common.Hash{0xaa}
	// Non-adjacent height and wrong parent => should return without popping
	payload := envelope(&eth.ExecutionPayload{BlockNumber: eth.Uint64Quantity(12), BlockHash: common.Hash{0x04}, ParentHash: common.Hash{0xff}})
	require.NoError(t, pq.Push(payload))

	ev := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    mkRef(10, headHash),
		SafeL2Head:      mkRef(9, common.Hash{0xbb}),
		FinalizedL2Head: mkRef(0, common.Hash{}),
	}

	pq.DropInapplicableUnsafePayloads(ev)
	require.Equal(t, 1, pq.Len())
	require.Equal(t, payload, pq.Peek())
}

func TestDropInapplicable_ApplicablePayloadKept(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	pq := NewPayloadsQueue(logger, payloadMemFixedCost*10, payloadMemSize)

	headHash := common.Hash{0xaa}
	// Correct parent and next height => should keep and break
	payload := envelope(&eth.ExecutionPayload{BlockNumber: eth.Uint64Quantity(11), BlockHash: common.Hash{0x05}, ParentHash: headHash})
	require.NoError(t, pq.Push(payload))

	ev := engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    mkRef(10, headHash),
		SafeL2Head:      mkRef(9, common.Hash{0xbb}),
		FinalizedL2Head: mkRef(0, common.Hash{}),
	}

	pq.DropInapplicableUnsafePayloads(ev)
	require.Equal(t, 1, pq.Len())
	require.Equal(t, payload, pq.Peek())
}
