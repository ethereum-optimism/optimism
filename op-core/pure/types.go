package pure

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1Input is a pre-processed L1 block containing only derivation-relevant data.
// The caller is responsible for filtering batcher transactions, extracting deposits
// from receipts, and extracting system config update logs.
type L1Input struct {
	Hash        common.Hash
	Number      uint64
	Timestamp   uint64
	BaseFee     *big.Int
	BlobBaseFee *big.Int
	ParentHash  common.Hash
	MixDigest   common.Hash // prevrandao

	BatcherData [][]byte         // raw batcher transaction data (calldata or blob content)
	Deposits    []*types.DepositTx
	ConfigLogs  []*types.Log // system config update logs, pre-filtered
}

// BlockRef converts L1Input header fields to an eth.L1BlockRef.
func (l *L1Input) BlockRef() eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       l.Hash,
		Number:     l.Number,
		ParentHash: l.ParentHash,
		Time:       l.Timestamp,
	}
}

// BlockID returns the block's ID (hash + number).
func (l *L1Input) BlockID() eth.BlockID {
	return eth.BlockID{Hash: l.Hash, Number: l.Number}
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

// l1InputInfo adapts L1Input to the eth.BlockInfo interface
// needed by derive.L1InfoDeposit.
type l1InputInfo struct {
	*L1Input
}

var _ eth.BlockInfo = (*l1InputInfo)(nil)

func (i *l1InputInfo) Hash() common.Hash        { return i.L1Input.Hash }
func (i *l1InputInfo) ParentHash() common.Hash   { return i.L1Input.ParentHash }
func (i *l1InputInfo) Coinbase() common.Address  { return common.Address{} }
func (i *l1InputInfo) Root() common.Hash         { return common.Hash{} }
func (i *l1InputInfo) NumberU64() uint64          { return i.L1Input.Number }
func (i *l1InputInfo) Time() uint64               { return i.L1Input.Timestamp }
func (i *l1InputInfo) MixDigest() common.Hash    { return i.L1Input.MixDigest }
func (i *l1InputInfo) BaseFee() *big.Int         { return i.L1Input.BaseFee }
func (i *l1InputInfo) ReceiptHash() common.Hash  { return common.Hash{} }
func (i *l1InputInfo) GasUsed() uint64            { return 0 }
func (i *l1InputInfo) GasLimit() uint64           { return 0 }
func (i *l1InputInfo) ParentBeaconRoot() *common.Hash { return nil }
func (i *l1InputInfo) WithdrawalsRoot() *common.Hash  { return nil }

func (i *l1InputInfo) BlobBaseFee(_ *params.ChainConfig) *big.Int {
	return i.L1Input.BlobBaseFee
}

func (i *l1InputInfo) ExcessBlobGas() *uint64 {
	if i.L1Input.BlobBaseFee != nil {
		zero := uint64(0)
		return &zero
	}
	return nil
}

func (i *l1InputInfo) BlobGasUsed() *uint64 { return nil }

func (i *l1InputInfo) HeaderRLP() ([]byte, error) {
	h := i.Header()
	return rlp.EncodeToBytes(h)
}

func (i *l1InputInfo) Header() *types.Header {
	return &types.Header{
		ParentHash: i.L1Input.ParentHash,
		Number:     new(big.Int).SetUint64(i.L1Input.Number),
		Time:       i.L1Input.Timestamp,
		BaseFee:    i.L1Input.BaseFee,
		MixDigest:  i.L1Input.MixDigest,
	}
}
