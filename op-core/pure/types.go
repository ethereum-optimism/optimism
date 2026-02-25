package pure

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
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
	return eth.L1BlockRef{
		Hash:       l.Header.Hash(),
		Number:     bigs.Uint64Strict(l.Header.Number),
		ParentHash: l.Header.ParentHash,
		Time:       l.Header.Time,
	}
}

// BlockID returns the block's ID (hash + number).
func (l *L1Input) BlockID() eth.BlockID {
	return eth.BlockID{Hash: l.Header.Hash(), Number: bigs.Uint64Strict(l.Header.Number)}
}

// blockInfo returns an eth.BlockInfo adapter for the L1Input's header,
// suitable for derive.L1InfoDeposit and similar consumers.
//
// L1InfoDeposit is called with a nil L1 chain config, so the standard
// HeaderBlockInfo cannot be used directly (its BlobBaseFee method calls
// eip4844.CalcBlobFee which requires a non-nil chain config). This wrapper
// delegates everything to HeaderBlockInfo except BlobBaseFee, which returns
// nil to let L1InfoDeposit apply its own fallback logic.
func (l *L1Input) blockInfo() eth.BlockInfo {
	return &l1BlockInfoAdapter{inner: eth.HeaderBlockInfo(l.Header)}
}

// l1BlockInfoAdapter wraps eth.BlockInfo to handle the nil-chainConfig case
// in BlobBaseFee. L1InfoDeposit computes blob base fee from ExcessBlobGas
// when BlobBaseFee returns nil, so we return nil here and let it handle the
// computation with its own chain config awareness.
type l1BlockInfoAdapter struct {
	inner eth.BlockInfo
}

var _ eth.BlockInfo = (*l1BlockInfoAdapter)(nil)

func (a *l1BlockInfoAdapter) Hash() common.Hash              { return a.inner.Hash() }
func (a *l1BlockInfoAdapter) ParentHash() common.Hash        { return a.inner.ParentHash() }
func (a *l1BlockInfoAdapter) Coinbase() common.Address       { return a.inner.Coinbase() }
func (a *l1BlockInfoAdapter) Root() common.Hash              { return a.inner.Root() }
func (a *l1BlockInfoAdapter) NumberU64() uint64              { return a.inner.NumberU64() }
func (a *l1BlockInfoAdapter) Time() uint64                   { return a.inner.Time() }
func (a *l1BlockInfoAdapter) MixDigest() common.Hash         { return a.inner.MixDigest() }
func (a *l1BlockInfoAdapter) BaseFee() *big.Int              { return a.inner.BaseFee() }
func (a *l1BlockInfoAdapter) ReceiptHash() common.Hash       { return a.inner.ReceiptHash() }
func (a *l1BlockInfoAdapter) GasUsed() uint64                { return a.inner.GasUsed() }
func (a *l1BlockInfoAdapter) GasLimit() uint64               { return a.inner.GasLimit() }
func (a *l1BlockInfoAdapter) ParentBeaconRoot() *common.Hash { return a.inner.ParentBeaconRoot() }
func (a *l1BlockInfoAdapter) WithdrawalsRoot() *common.Hash  { return a.inner.WithdrawalsRoot() }
func (a *l1BlockInfoAdapter) ExcessBlobGas() *uint64         { return a.inner.ExcessBlobGas() }
func (a *l1BlockInfoAdapter) BlobGasUsed() *uint64           { return a.inner.BlobGasUsed() }
func (a *l1BlockInfoAdapter) HeaderRLP() ([]byte, error)     { return a.inner.HeaderRLP() }
func (a *l1BlockInfoAdapter) Header() *types.Header          { return a.inner.Header() }

// BlobBaseFee computes the blob base fee from the header's ExcessBlobGas
// without requiring an L1 chain config. L1InfoDeposit is called with a nil
// L1 chain config in pure derivation, so we cannot delegate to the standard
// HeaderBlockInfo.BlobBaseFee (which calls eip4844.CalcBlobFee with the
// chain config). Instead we use CalcBlobFeeCancun which only needs the
// excess blob gas value.
func (a *l1BlockInfoAdapter) BlobBaseFee(_ *params.ChainConfig) *big.Int {
	ebg := a.inner.ExcessBlobGas()
	if ebg == nil {
		return nil
	}
	return eth.CalcBlobFeeCancun(*ebg)
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
