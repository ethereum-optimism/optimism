package batcher

import (
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/core/types"
)

type SizedBlock struct {
	*types.Block
	rawSize          uint64
	estimatedDABytes uint64
}

func ToSizedBlock(block *types.Block) SizedBlock {
	b := SizedBlock{Block: block}
	// populate caches
	b.RawSize()
	b.EstimatedDABytes()
	return b
}

func (b *SizedBlock) RawSize() uint64 {
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

func (b *SizedBlock) EstimatedDABytes() uint64 {
	if b.estimatedDABytes == 0 {
		daSize := uint64(70) // estimated overhead of batch metadata
		for _, tx := range b.Transactions() {
			// Deposit transactions are not included in batches
			if tx.IsDepositTx() {
				continue
			}
			// It is safe to assume that the estimated DA size is always a uint64
			daSize += bigs.Uint64Strict(tx.RollupCostData().EstimatedDASize())
		}
		b.estimatedDABytes = daSize
	}
	return b.estimatedDABytes
}
