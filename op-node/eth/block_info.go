package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
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
	ReceiptHash() common.Hash
	GasUsed() uint64

	// HeaderRLP returns the RLP of the block header as per consensus rules
	// Returns an error if the header RLP could not be written
	HeaderRLP() ([]byte, error)
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

func (b blockInfo) HeaderRLP() ([]byte, error) {
	return rlp.EncodeToBytes(b.Header())
}

func BlockToInfo(b *types.Block) BlockInfo {
	return blockInfo{b}
}

var _ BlockInfo = (*blockInfo)(nil)

// headerBlockInfo is a conversion type of types.Header turning it into a
// BlockInfo.
type headerBlockInfo struct{ *types.Header }

func (h headerBlockInfo) ParentHash() common.Hash {
	return h.Header.ParentHash
}

func (h headerBlockInfo) Coinbase() common.Address {
	return h.Header.Coinbase
}

func (h headerBlockInfo) Root() common.Hash {
	return h.Header.Root
}

func (h headerBlockInfo) NumberU64() uint64 {
	return h.Header.Number.Uint64()
}

func (h headerBlockInfo) Time() uint64 {
	return h.Header.Time
}

func (h headerBlockInfo) MixDigest() common.Hash {
	return h.Header.MixDigest
}

func (h headerBlockInfo) BaseFee() *big.Int {
	return h.Header.BaseFee
}

func (h headerBlockInfo) ReceiptHash() common.Hash {
	return h.Header.ReceiptHash
}

func (h headerBlockInfo) GasUsed() uint64 {
	return h.Header.GasUsed
}

func (h headerBlockInfo) HeaderRLP() ([]byte, error) {
	return rlp.EncodeToBytes(h.Header)
}

// HeaderBlockInfo returns h as a BlockInfo implementation.
func HeaderBlockInfo(h *types.Header) BlockInfo {
	return headerBlockInfo{h}
}
