package eth

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type BlockInfo interface {
	Hash() common.Hash
	ParentHash() common.Hash
	Coinbase() common.Address
	Root() common.Hash // state-root
	NumberU64() uint64
	Time() uint64
	// MixDigest field, reused for randomness after The Merge (Bellatrix hardfork)
	MixDigest() common.Hash
	BaseFee() *big.Int
	// BlobBaseFee returns the blob base fee computed from the excess blob gas, or nil if the block
	// has no excess blob gas. It returns an error if the block has excess blob gas but chainConfig
	// has no blob schedule active at the block time.
	BlobBaseFee(chainConfig *params.ChainConfig) (*big.Int, error)
	ExcessBlobGas() *uint64
	ReceiptHash() common.Hash
	GasUsed() uint64
	BlobGasUsed() *uint64
	GasLimit() uint64
	ParentBeaconRoot() *common.Hash // Dencun extension
	WithdrawalsRoot() *common.Hash  // Isthmus extension
	Extra() []byte
}

func InfoToL1BlockRef(info BlockInfo) L1BlockRef {
	return L1BlockRef{
		Hash:       info.Hash(),
		Number:     info.NumberU64(),
		ParentHash: info.ParentHash(),
		Time:       info.Time(),
	}
}

type NumberAndHash interface {
	Hash() common.Hash
	NumberU64() uint64
}

func ToBlockID(b NumberAndHash) BlockID {
	return BlockID{
		Hash:   b.Hash(),
		Number: b.NumberU64(),
	}
}

// blockInfo is a conversion type of types.Block turning it into a BlockInfo
type blockInfo struct{ *types.Block }

func (b blockInfo) BlobBaseFee(chainConfig *params.ChainConfig) (*big.Int, error) {
	return HeaderBlobBaseFee(chainConfig, b.Block.Header())
}

func (b blockInfo) ParentBeaconRoot() *common.Hash {
	return b.Block.BeaconRoot()
}

func (b blockInfo) WithdrawalsRoot() *common.Hash {
	return b.Block.Header().WithdrawalsHash
}

func BlockToInfo(b *types.Block) BlockInfo {
	return blockInfo{b}
}

var _ BlockInfo = (*blockInfo)(nil)

// headerBlockInfo is a conversion type of types.Header turning it into a
// BlockInfo, but using a cached hash value.
type headerBlockInfo struct {
	hash   common.Hash
	header *types.Header
}

var _ BlockInfo = (*headerBlockInfo)(nil)

func (h *headerBlockInfo) Hash() common.Hash {
	return h.hash
}

func (h *headerBlockInfo) ParentHash() common.Hash {
	return h.header.ParentHash
}

func (h *headerBlockInfo) Coinbase() common.Address {
	return h.header.Coinbase
}

func (h *headerBlockInfo) Root() common.Hash {
	return h.header.Root
}

func (h *headerBlockInfo) NumberU64() uint64 {
	return bigs.Uint64Strict(h.header.Number)
}

func (h *headerBlockInfo) Time() uint64 {
	return h.header.Time
}

func (h *headerBlockInfo) MixDigest() common.Hash {
	return h.header.MixDigest
}

func (h *headerBlockInfo) BaseFee() *big.Int {
	return h.header.BaseFee
}

func (h *headerBlockInfo) BlobBaseFee(chainConfig *params.ChainConfig) (*big.Int, error) {
	return HeaderBlobBaseFee(chainConfig, h.header)
}

func (h *headerBlockInfo) ExcessBlobGas() *uint64 {
	return h.header.ExcessBlobGas
}

func (h *headerBlockInfo) ReceiptHash() common.Hash {
	return h.header.ReceiptHash
}

func (h *headerBlockInfo) GasUsed() uint64 {
	return h.header.GasUsed
}

func (h *headerBlockInfo) BlobGasUsed() *uint64 {
	return h.header.BlobGasUsed
}

func (h *headerBlockInfo) GasLimit() uint64 {
	return h.header.GasLimit
}

func (h *headerBlockInfo) ParentBeaconRoot() *common.Hash {
	return h.header.ParentBeaconRoot
}

func (h *headerBlockInfo) WithdrawalsRoot() *common.Hash {
	return h.header.WithdrawalsHash
}

func (h *headerBlockInfo) Extra() []byte {
	return h.header.Extra
}

func (h *headerBlockInfo) MarshalJSON() ([]byte, error) {
	return h.header.MarshalJSON()
}

func (h *headerBlockInfo) UnmarshalJSON(input []byte) error {
	return h.header.UnmarshalJSON(input)
}

// maxExcessBlobGas is the largest excess blob gas HeaderBlobBaseFee accepts. The cost of computing
// the blob fee grows with the square of the excess blob gas. At this bound, the blob fee is above
// e^200 wei under every blob schedule go-ethereum defines, which no L1 block can reach.
const maxExcessBlobGas = 1 << 32

// HeaderBlobBaseFee returns the blob base fee of h computed from its excess blob gas, or nil if h
// has no excess blob gas. It returns an error if h has excess blob gas but chainConfig has no blob
// schedule active at the time of h, or if the excess blob gas is implausibly large.
func HeaderBlobBaseFee(chainConfig *params.ChainConfig, h *types.Header) (*big.Int, error) {
	if h.ExcessBlobGas == nil {
		return nil, nil
	}
	// eip4844.CalcBlobFee panics if there is no blob schedule at the header time, and
	// MaxBlobsPerBlock is 0 then.
	if eip4844.MaxBlobsPerBlock(chainConfig, h.Time) == 0 {
		return nil, fmt.Errorf("header %v has excess blob gas, but no blob schedule is active at its time %d", h.Number, h.Time)
	}
	if *h.ExcessBlobGas > maxExcessBlobGas {
		return nil, fmt.Errorf("header %v has excess blob gas %d above maximum %d", h.Number, *h.ExcessBlobGas, maxExcessBlobGas)
	}
	return eip4844.CalcBlobFee(chainConfig, h), nil
}

// HeaderBlockInfo returns h as a BlockInfo implementation, with pre-cached blockhash.
func HeaderBlockInfo(h *types.Header) BlockInfo {
	return &headerBlockInfo{hash: h.Hash(), header: h}
}

// HeaderBlockInfoTrusted returns a BlockInfo, with trusted pre-cached block-hash.
func HeaderBlockInfoTrusted(hash common.Hash, h *types.Header) BlockInfo {
	return &headerBlockInfo{hash: hash, header: h}
}
