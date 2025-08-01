package batcher

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockWithDABytes struct {
	*types.Block
	rawSize          uint64
	estimatedDABytes uint64
}

func ToBlockWithDABytes(block *types.Block) BlockWithDABytes {
	b := BlockWithDABytes{Block: block}
	// populate caches
	b.RawSize()
	b.EstimatedDABytes()
	return b
}

func (b *BlockWithDABytes) RawSize() uint64 {
	if b.rawSize == 0 {
		b.rawSize = uint64(70)
		for _, tx := range b.Transactions() {
			// Deposit transactions are not included in batches
			if tx.IsDepositTx() {
				continue
			}
			// Add 2 for the overhead of encoding the tx bytes in a RLP list
			b.rawSize += tx.Size() + 2
		}
	}
	return b.rawSize
}

func (b *BlockWithDABytes) EstimatedDABytes() uint64 {
	if b.estimatedDABytes == 0 {
		daSize := uint64(70) // estimated overhead of batch metadata
		for _, tx := range b.Transactions() {
			// Deposit transactions are not included in batches
			if tx.IsDepositTx() {
				continue
			}
			bigSize := tx.RollupCostData().EstimatedDASize()
			if bigSize.IsUint64() { // this should always be true, but if not just ignore
				daSize += bigSize.Uint64()
			}
		}
		b.estimatedDABytes = daSize
	}
	return b.estimatedDABytes
}
