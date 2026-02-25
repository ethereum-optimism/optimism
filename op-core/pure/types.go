package pure

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1Input is a pre-processed L1 block containing only derivation-relevant data.
// The caller is responsible for filtering batcher transactions, extracting deposits
// from receipts, and extracting system config update logs.
//
// Header contains the full L1 block header. Callers will typically already have
// the header at hand when constructing an L1Input.
type L1Input struct {
	Header *types.Header

	BatcherData [][]byte           // raw batcher transaction data (calldata or blob content)
	Deposits    []*types.DepositTx // user deposit transactions extracted from receipts
	ConfigLogs  []*types.Log       // system config update logs, pre-filtered
}

// BlockRef converts the L1 header to an eth.L1BlockRef.
func (l *L1Input) BlockRef() eth.L1BlockRef {
	return *eth.BlockRefFromHeader(l.Header)
}

// BlockID returns the block's ID (hash + number).
func (l *L1Input) BlockID() eth.BlockID {
	return eth.HeaderBlockID(l.Header)
}

// DerivedBlock is a single derived L2 block -- payload attributes ready for execution.
type DerivedBlock struct {
	Attributes         *eth.PayloadAttributes
	ExpectedParentHash common.Hash // from batch ParentHash field; zero if unavailable
	DerivedFrom        eth.L1BlockRef
}

// l2Cursor tracks the derivation position without knowing the L2 block hash.
type l2Cursor struct {
	Number         uint64
	Timestamp      uint64
	L1Origin       eth.BlockID
	SequenceNumber uint64
}

func newCursor(safeHead eth.L2BlockRef) l2Cursor {
	return l2Cursor{
		Number:         safeHead.Number,
		Timestamp:      safeHead.Time,
		L1Origin:       safeHead.L1Origin,
		SequenceNumber: safeHead.SequenceNumber,
	}
}

func (c *l2Cursor) advance(timestamp uint64, l1Origin eth.BlockID, seqNum uint64) {
	c.Number++
	c.Timestamp = timestamp
	c.L1Origin = l1Origin
	c.SequenceNumber = seqNum
}

// needsEmptyBatch returns true when the sequencing window has expired,
// meaning the cursor's L1 origin is more than SeqWindowSize blocks behind
// the current L1 block.
func (c l2Cursor) needsEmptyBatch(currentL1 eth.L1BlockRef, cfg *rollup.Config) bool {
	return currentL1.Number > c.L1Origin.Number+cfg.SeqWindowSize
}
